package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

// ErrTooManyRedirects is returned (wrapped) by Get when the response chain
// exceeds Options.MaxRedirects.
var ErrTooManyRedirects = errors.New("too many redirects")

// Options controls per-request limits. Zero values fall back to defaults.
type Options struct {
	Timeout      time.Duration
	MaxBytes     int64
	MaxRedirects int
}

// Response is the result of a successful Get.
type Response struct {
	Body        []byte
	ContentType string
	Truncated   bool
	// Markdown is set when ContentType is text/html: the body converted to
	// Markdown via HTMLToMarkdown. Empty for non-HTML responses.
	Markdown string
}

func defaultOptions() Options {
	return Options{
		Timeout:      30 * time.Second,
		MaxBytes:     5 * 1024 * 1024,
		MaxRedirects: 10,
	}
}

func mergeOptions(o Options) Options {
	d := defaultOptions()
	if o.Timeout > 0 {
		d.Timeout = o.Timeout
	}
	if o.MaxBytes > 0 {
		d.MaxBytes = o.MaxBytes
	}
	if o.MaxRedirects > 0 {
		d.MaxRedirects = o.MaxRedirects
	}
	return d
}

// Get issues an HTTP GET against url, enforcing timeout, size, and redirect
// caps from opts. The returned Response.Body is at most opts.MaxBytes long;
// when the upstream body exceeded that, Response.Truncated is true.
func Get(ctx context.Context, url string, opts Options) (*Response, error) {
	o := mergeOptions(opts)

	client := &http.Client{
		Timeout: o.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= o.MaxRedirects {
				return ErrTooManyRedirects
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, ErrTooManyRedirects) {
			return nil, fmt.Errorf("fetch: %w", ErrTooManyRedirects)
		}
		return nil, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, o.MaxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	truncated := false
	if int64(len(body)) > o.MaxBytes {
		body = body[:o.MaxBytes]
		truncated = true
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		if mt, _, err := mime.ParseMediaType(contentType); err == nil {
			contentType = mt
		} else {
			contentType = strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
		}
	}

	var markdown string
	if strings.HasPrefix(contentType, "text/html") {
		md, err := HTMLToMarkdown(body, url)
		if err != nil {
			return nil, fmt.Errorf("fetch: html to markdown: %w", err)
		}
		markdown = md
	}

	return &Response{
		Body:        body,
		ContentType: contentType,
		Truncated:   truncated,
		Markdown:    markdown,
	}, nil
}
