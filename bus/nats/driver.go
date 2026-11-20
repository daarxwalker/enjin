package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/daarxwalker/enjin/bus"

	natssdk "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var _ bus.Driver = (*Driver)(nil)

type Driver struct {
	conn   *natssdk.Conn
	js     jetstream.JetStream
	log    *slog.Logger
	prefix string
	stream string
	maxAge time.Duration
}

const maxDeliver = 5

func NewDriver(ctx context.Context, conn *natssdk.Conn, options ...Option) (*Driver, error) {
	d := &Driver{
		conn:   conn,
		log:    slog.Default(),
		prefix: "bus",
	}
	for _, option := range options {
		option(d)
	}
	if d.stream == "" {
		return d, nil
	}
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("bus: create jetstream failed: %w", err)
	}
	if _, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      d.stream,
		Subjects:  []string{d.prefix + ".>"},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    d.maxAge,
	}); err != nil {
		return nil, fmt.Errorf("bus: create stream %s failed: %w", d.stream, err)
	}
	d.js = js
	return d, nil
}

func (d *Driver) Publish(ctx context.Context, subject string, data []byte) error {
	if d.js == nil {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("bus: publish %s stopped: %w", subject, err)
		}
		if err := d.publish(subject, data); err != nil {
			return fmt.Errorf("bus: publish %s failed: %w", subject, err)
		}
		return nil
	}
	if _, err := d.js.Publish(ctx, d.subject(subject), data); err != nil {
		return fmt.Errorf("bus: publish %s failed: %w", subject, err)
	}
	return nil
}

func (d *Driver) TryPublish(subject string, data []byte) error {
	if d.js == nil {
		if err := d.publish(subject, data); err != nil {
			return fmt.Errorf("bus: try publish %s failed: %w", subject, err)
		}
		return nil
	}
	_, err := d.js.PublishAsync(d.subject(subject), data)
	if errors.Is(err, jetstream.ErrTooManyStalledMsgs) {
		return fmt.Errorf("bus: try publish %s failed: %w", subject, bus.ErrFull)
	}
	if err != nil {
		return fmt.Errorf("bus: publish %s failed: %w", subject, err)
	}
	return nil
}

func (d *Driver) Subscribe(
	subject, queue string,
	handler func(ctx context.Context, data []byte) error,
) (bus.Subscription, error) {
	var (
		subscription bus.Subscription
		err          error
	)
	if d.js == nil {
		subscription, err = d.subscribe(subject, queue, handler)
	} else {
		subscription, err = d.consume(subject, queue, handler)
	}
	if err != nil {
		return nil, fmt.Errorf("bus: subscribe %s failed: %w", subject, err)
	}
	return subscription, nil
}

func (d *Driver) publish(subject string, data []byte) error {
	err := d.conn.Publish(d.subject(subject), data)
	if errors.Is(err, natssdk.ErrReconnectBufExceeded) {
		return fmt.Errorf("bus: publish %s failed: %w", subject, bus.ErrFull)
	}
	if err != nil {
		return fmt.Errorf("bus: publish %s failed: %w", subject, err)
	}
	return nil
}

func (d *Driver) subscribe(
	subject, queue string,
	handler func(ctx context.Context, data []byte) error,
) (bus.Subscription, error) {
	sub, err := d.conn.QueueSubscribe(d.subject(subject), queue, func(msg *natssdk.Msg) {
		if handleErr := handler(context.Background(), msg.Data); handleErr != nil {
			d.log.Error(fmt.Sprintf("bus: handle %s failed: %s", subject, handleErr))
		}
	})
	if err != nil {
		d.log.Error(fmt.Sprintf("bus: subscribe %s failed: %s", subject, err))
		return nil, fmt.Errorf("bus: subscribe %s failed: %w", subject, err)
	}
	return &natsSubscription{sub: sub}, nil
}

func (d *Driver) consume(
	subject, queue string,
	handler func(ctx context.Context, data []byte) error,
) (bus.Subscription, error) {
	consumer, err := d.createConsumer(context.Background(), subject, queue)
	if err != nil {
		d.log.Error(fmt.Sprintf("bus: create consumer %s failed: %s", subject, err))
		return nil, fmt.Errorf("bus: create consumer %s failed: %w", subject, err)
	}
	consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		d.handle(msg, subject, queue != "", handler)
	})
	if err != nil {
		d.log.Error(fmt.Sprintf("bus: consume %s failed: %s", subject, err))
		return nil, fmt.Errorf("bus: consume %s failed: %w", subject, err)
	}
	return &jetStreamSubscription{consumeCtx: consumeCtx}, nil
}

func (d *Driver) createConsumer(ctx context.Context, subject, queue string) (jetstream.Consumer, error) {
	if queue == "" {
		consumer, err := d.js.OrderedConsumer(ctx, d.stream, jetstream.OrderedConsumerConfig{
			FilterSubjects: []string{d.subject(subject)},
			DeliverPolicy:  jetstream.DeliverNewPolicy,
		})
		if err != nil {
			return nil, fmt.Errorf("bus: create ordered consumer for %s failed: %w", subject, err)
		}
		return consumer, nil
	}
	consumer, err := d.js.CreateOrUpdateConsumer(ctx, d.stream, jetstream.ConsumerConfig{
		Durable:       d.durable(subject, queue),
		FilterSubject: d.subject(subject),
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		MaxDeliver:    maxDeliver,
	})
	if err != nil {
		return nil, fmt.Errorf("bus: create durable consumer for %s in queue %s failed: %w", subject, queue, err)
	}
	return consumer, nil
}

func (d *Driver) handle(
	msg jetstream.Msg,
	subject string,
	ack bool,
	handler func(ctx context.Context, data []byte) error,
) {
	if err := handler(context.Background(), msg.Data()); err != nil {
		d.log.Error(fmt.Sprintf("bus: handle %s failed: %s", subject, err))
		if ack {
			if nakErr := msg.Nak(); nakErr != nil {
				d.log.Error(fmt.Sprintf("bus: nak %s failed: %s", subject, nakErr))
			}
		}
		return
	}
	if ack {
		if ackErr := msg.Ack(); ackErr != nil {
			d.log.Error(fmt.Sprintf("bus: ack %s failed: %s", subject, ackErr))
		}
	}
}

func (d *Driver) subject(subject string) string {
	return d.prefix + "." + subject
}

func (d *Driver) durable(subject, queue string) string {
	return strings.NewReplacer(".", "_", "*", "_", ">", "_", " ", "_").Replace(queue + "_" + d.subject(subject))
}

type natsSubscription struct {
	sub *natssdk.Subscription
}

func (s *natsSubscription) Close(ctx context.Context) error {
	closed := s.sub.StatusChanged(natssdk.SubscriptionClosed)
	if err := s.sub.Drain(); err != nil {
		return fmt.Errorf("bus: drain %s failed: %w", s.sub.Subject, err)
	}
	select {
	case <-closed:
		return nil
	case <-ctx.Done():
		drainErr := fmt.Errorf("bus: drain subscription %s stopped: %w", s.sub.Subject, ctx.Err())
		if err := s.sub.Unsubscribe(); err != nil && !errors.Is(err, natssdk.ErrBadSubscription) {
			return errors.Join(drainErr, fmt.Errorf("bus: unsubscribe %s failed: %w", s.sub.Subject, err))
		}
		return drainErr
	}
}

type jetStreamSubscription struct {
	consumeCtx jetstream.ConsumeContext
}

func (s *jetStreamSubscription) Close(ctx context.Context) error {
	s.consumeCtx.Drain()
	select {
	case <-s.consumeCtx.Closed():
		return nil
	case <-ctx.Done():
		s.consumeCtx.Stop()
		return fmt.Errorf("bus: drain jetstream consumer stopped: %w", ctx.Err())
	}
}
