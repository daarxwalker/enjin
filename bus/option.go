package bus

type Option func(*route)

func WithSubject(subject string) Option {
	return func(r *route) {
		r.subject = subject
	}
}

func WithQueue(queue string) Option {
	return func(r *route) {
		r.queue = queue
	}
}
