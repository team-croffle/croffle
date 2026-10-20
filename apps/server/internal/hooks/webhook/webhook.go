// Package webhook delivers hook events to an HTTP endpoint.
//
// It is the one sink server 0.1 ships, and it exists to prove the seam works
// without putting a code-execution surface inside the process: whatever an
// operator wants to do with "a schedule was shared" happens in their own
// service, not in this binary.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/hooks"
)

// Headers a receiver reads.
const (
	HeaderEvent     = "X-Croffle-Event"
	HeaderSignature = "X-Croffle-Signature"
	HeaderDelivery  = "X-Croffle-Delivery"
)

const (
	deliveryTimeout = 5 * time.Second
	maxAttempts     = 3
)

// retryDelays are the waits between attempts. Short: a hook is a notification,
// and holding the dispatcher for minutes would drop later events instead.
var retryDelays = []time.Duration{time.Second, 5 * time.Second}

// Config is the operator's webhook setup, read from the environment.
type Config struct {
	URL    string
	Secret string
	// Types to deliver. Empty means hooks.DefaultTypes.
	Types []string
}

// Sink posts events to Config.URL.
type Sink struct {
	url    string
	secret string
	types  []string
	client *http.Client
	logger *slog.Logger
	sleep  func(time.Duration)
}

// New validates the configuration and builds the sink. It returns nil when no
// URL is configured, which is the normal case.
func New(cfg Config, logger *slog.Logger) (*Sink, error) {
	target := strings.TrimSpace(cfg.URL)
	if target == "" {
		return nil, nil
	}

	parsed, err := url.Parse(target)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("webhook url %q: want an http(s) URL", cfg.URL)
	}
	if strings.TrimSpace(cfg.Secret) == "" {
		// Refusing rather than sending unsigned: a receiver that cannot verify
		// the sender has to trust anyone who knows the URL.
		return nil, errors.New("a webhook url needs a secret, so the receiver can verify the sender")
	}

	types := cfg.Types
	if len(types) == 0 {
		types = hooks.DefaultTypes
	}
	for _, name := range types {
		if !slices.Contains(hooks.AllTypes, name) {
			return nil, fmt.Errorf("unknown webhook event %q (known: %s)",
				name, strings.Join(hooks.AllTypes, ", "))
		}
	}

	if logger == nil {
		logger = slog.Default()
	}
	return &Sink{
		url:    target,
		secret: cfg.Secret,
		types:  types,
		client: &http.Client{Timeout: deliveryTimeout},
		logger: logger,
		sleep:  time.Sleep,
	}, nil
}

func (s *Sink) Name() string { return "webhook" }

// Types reports what this sink delivers, for the startup log.
func (s *Sink) Types() []string { return s.types }

// Deliver posts one event, retrying a network failure or a 5xx. A 4xx is the
// receiver saying "do not send me this", so it is not retried.
func (s *Sink) Deliver(ctx context.Context, event hooks.Event) error {
	if !slices.Contains(s.types, event.Type) {
		return nil
	}

	body, err := hooks.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}

	signature := sign(s.secret, body)
	delivery := newDeliveryID()

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			s.sleep(retryDelays[min(attempt-1, len(retryDelays)-1)])
		}

		status, err := s.post(ctx, event.Type, body, signature, delivery)
		switch {
		case err == nil && status < 300:
			return nil
		case err == nil && status < 500:
			// The receiver rejected it deliberately.
			return fmt.Errorf("webhook %s: receiver answered %d", event.Type, status)
		case err == nil:
			lastErr = fmt.Errorf("receiver answered %d", status)
		default:
			lastErr = err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return fmt.Errorf("webhook %s after %d attempts: %w", event.Type, maxAttempts, lastErr)
}

func (s *Sink) post(ctx context.Context, eventType string, body []byte, signature, delivery string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderEvent, eventType)
	req.Header.Set(HeaderSignature, signature)
	req.Header.Set(HeaderDelivery, delivery)

	res, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = res.Body.Close() }()
	// Drained so the connection can be reused.
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))

	return res.StatusCode, nil
}

// sign is the HMAC a receiver checks, in the same shape GitHub uses so the
// verification snippet people already have works.
func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify is the receiver side, exported so the documentation can point at a
// real implementation instead of describing one.
func Verify(secret string, body []byte, signature string) bool {
	return hmac.Equal([]byte(sign(secret, body)), []byte(signature))
}

func newDeliveryID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
