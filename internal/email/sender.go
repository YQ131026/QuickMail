package email

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"QuickMail/internal/config"
	gomail "gopkg.in/gomail.v2"
)

var ErrAllProvidersFailed = errors.New("all providers failed")

// Attachment represents an email attachment payload.
type Attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// SendRequest captures the payload to send an email.
type SendRequest struct {
	Subject          string       `json:"subject"`
	Body             string       `json:"body"`
	IsHTML           bool         `json:"is_html"`
	To               []string     `json:"to"`
	Cc               []string     `json:"cc"`
	Bcc              []string     `json:"bcc"`
	Attachments      []Attachment `json:"attachments"`
	ProviderPriority []string     `json:"provider_priority"`
	From             string       `json:"from"`
}

// Sender handles dispatching emails using configured providers.
type Sender struct {
	Store  *config.Store
	Logger *log.Logger
}

func (s *Sender) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
	}
}

func (s *Sender) Send(ctx context.Context, req SendRequest) error {
	if len(req.To) == 0 {
		return errors.New("at least one recipient is required")
	}
	if strings.TrimSpace(req.Subject) == "" {
		return errors.New("subject is required")
	}

	priority := req.ProviderPriority
	if len(priority) == 0 {
		providers, err := s.Store.ListProviders()
		if err != nil {
			return err
		}
		for _, p := range providers {
			priority = append(priority, p.Name)
		}
	}

	if len(priority) == 0 {
		return errors.New("no providers configured")
	}

	var attempts []string
	for _, name := range priority {
		provider, err := s.Store.GetProvider(name)
		if err != nil {
			attempts = append(attempts, fmt.Sprintf("%s:error:%v", name, err))
			continue
		}

		if err := s.sendWithProvider(ctx, provider, req); err != nil {
			attempts = append(attempts, fmt.Sprintf("%s:error:%v", name, err))
			s.logf("send failed using provider %s: %v", name, err)
			continue
		}

		s.logf("email sent successfully via provider %s", name)
		return nil
	}

	return fmt.Errorf("%w: %s", ErrAllProvidersFailed, strings.Join(attempts, "; "))
}

func (s *Sender) sendWithProvider(ctx context.Context, provider config.Provider, req SendRequest) error {
	message := gomail.NewMessage()

	from := req.From
	if from == "" {
		from = provider.From
	}
	if from == "" {
		return errors.New("sender address is required (missing both request and provider 'from')")
	}

	message.SetHeader("From", from)
	message.SetHeader("To", req.To...)
	if len(req.Cc) > 0 {
		message.SetHeader("Cc", req.Cc...)
	}
	if len(req.Bcc) > 0 {
		message.SetHeader("Bcc", req.Bcc...)
	}

	contentType := "text/plain"
	if req.IsHTML {
		contentType = "text/html"
	}
	message.SetBody(contentType, req.Body)

	for _, att := range req.Attachments {
		if att.Filename == "" || att.Content == "" {
			return errors.New("attachment requires filename and base64 content")
		}

		data, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			return fmt.Errorf("decode attachment %s: %w", att.Filename, err)
		}

		opts := []gomail.FileSetting{gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(data)
			return err
		})}

		if att.ContentType != "" {
			opts = append(opts, gomail.SetHeader(map[string][]string{
				"Content-Type": {att.ContentType},
			}))
		}

		message.Attach(att.Filename, opts...)
	}

	dialer := gomail.NewDialer(provider.Host, provider.Port, provider.Username, provider.Password)
	dialer.SSL = provider.UseTLS

	ctx, cancel := withTimeout(ctx, 15*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- dialer.DialAndSend(message)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// CheckProvider tests the SMTP connection for a provider.
func (s *Sender) CheckProvider(ctx context.Context, name string) error {
	provider, err := s.Store.GetProvider(name)
	if err != nil {
		return err
	}

	dialer := gomail.NewDialer(provider.Host, provider.Port, provider.Username, provider.Password)
	dialer.SSL = provider.UseTLS

	ctx, cancel := withTimeout(ctx, 10*time.Second)
	defer cancel()

	type result struct {
		sc  gomail.SendCloser
		err error
	}

	ch := make(chan result, 1)
	go func() {
		sc, err := dialer.Dial()
		ch <- result{sc: sc, err: err}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return res.err
		}
		return res.sc.Close()
	}
}

func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
