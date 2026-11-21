package antibot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/web/theme"

	"github.com/a-h/templ"
)

type TurnstileDriver struct {
	client          *http.Client
	siteKey, secret string
}

type turnstileResponse struct {
	Errors  []string `json:"error-codes"`
	Success bool     `json:"success"`
}

const (
	turnstileFormValueName = "cf-turnstile-response"
)

// Turnstile calls the theme that follows the browser "auto".
const turnstileSystemTheme = "auto"

const (
	turnstileScriptURL            = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit"
	turnstileVerificationEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	turnstileTimeout              = 5 * time.Second
)

func NewTurnstileDriver(client *http.Client, siteKey, secret string) *TurnstileDriver {
	m := &TurnstileDriver{
		client:  client,
		siteKey: siteKey,
		secret:  secret,
	}
	return m
}

func (m *TurnstileDriver) CreateWidget(ctx context.Context) templ.Component {
	ctxTheme := theme.From(ctx)
	currentTheme := ctxTheme.String()
	if ctxTheme == theme.System {
		currentTheme = turnstileSystemTheme
	}
	return TurnstileWidget(m.siteKey, currentTheme) //nolint:contextcheck // templ passes ctx to Render
}

func (m *TurnstileDriver) Verify(r *http.Request) (success bool, err error) {
	var zero bool

	token := r.FormValue(turnstileFormValueName)
	if token == "" {
		return zero, errors.New("cf turnstile no token provided")
	}
	ok, verifyErr := m.callVerificationEndpoint(r.Context(), m.secret, token, httpx.ClientIP(r))
	if verifyErr != nil {
		return zero, fmt.Errorf("cf turnstile verify token failed: %w", verifyErr)
	}
	if !ok {
		return zero, errors.New("cf turnstile token rejected")
	}
	return true, nil
}

func (m *TurnstileDriver) callVerificationEndpoint(
	ctx context.Context,
	secret, token, remoteIP string,
) (success bool, err error) {
	var zero bool

	ctx, cancel := context.WithTimeout(ctx, turnstileTimeout)
	defer cancel()

	data := make(url.Values)
	data.Set("secret", secret)
	data.Set("response", token)
	data.Set("remoteip", remoteIP)

	req, reqErr := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		turnstileVerificationEndpoint,
		strings.NewReader(data.Encode()),
	)
	if reqErr != nil {
		return zero, fmt.Errorf("create cf turnstile verification request failed: %w", reqErr)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, verifyErr := m.client.Do(req)
	if verifyErr != nil {
		return zero, fmt.Errorf("cf turnstile verification failed: %w", verifyErr)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close cf turnstile verification body failed: %w", closeErr))
		}
	}()

	if res.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("cf turnstile verification failed with status code %d", res.StatusCode)
	}

	var result turnstileResponse
	if decodeErr := json.NewDecoder(res.Body).Decode(&result); decodeErr != nil {
		return zero, fmt.Errorf("decode cf turnstile verification response failed: %w", decodeErr)
	}

	if len(result.Errors) > 0 {
		var errs []error
		for _, e := range result.Errors {
			errs = append(errs, errors.New(e))
		}
		return result.Success, fmt.Errorf("cf turnstile verification reported errors: %w", errors.Join(errs...))
	}

	return result.Success, nil
}
