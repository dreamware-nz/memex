package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTMLToMarkdown_BasicAndRelativeLink(t *testing.T) {
	html := []byte(`<h1>Hello</h1><p>World</p><a href="/rel">link</a>`)

	md, err := HTMLToMarkdown(html, "https://example.com")
	if err != nil {
		t.Fatalf("HTMLToMarkdown: %v", err)
	}
	if !strings.Contains(md, "Hello") {
		t.Errorf("missing heading text in %q", md)
	}
	if !strings.Contains(md, "World") {
		t.Errorf("missing paragraph text in %q", md)
	}
	if !strings.Contains(md, "https://example.com/rel") {
		t.Errorf("relative link not resolved: %q", md)
	}
}

func TestHTMLToMarkdown_Empty(t *testing.T) {
	md, err := HTMLToMarkdown(nil, "https://example.com")
	if err != nil {
		t.Fatalf("HTMLToMarkdown(nil): %v", err)
	}
	if strings.TrimSpace(md) != "" {
		t.Errorf("expected empty, got %q", md)
	}

	md, err = HTMLToMarkdown([]byte(""), "")
	if err != nil {
		t.Fatalf("HTMLToMarkdown(empty): %v", err)
	}
	if strings.TrimSpace(md) != "" {
		t.Errorf("expected empty, got %q", md)
	}
}

func TestHTMLToMarkdown_Malformed(t *testing.T) {
	md, err := HTMLToMarkdown([]byte("<h1>oops<p>still ok"), "")
	if err != nil {
		t.Fatalf("malformed HTML returned error: %v", err)
	}
	if strings.TrimSpace(md) == "" {
		t.Errorf("expected best-effort markdown, got empty")
	}
}

func TestGet_HTMLConvertedToMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<h1>Hi</h1><p>Body</p>`))
	}))
	defer srv.Close()

	resp, err := Get(context.Background(), srv.URL, Options{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.ContentType != "text/html" {
		t.Fatalf("ContentType = %q, want text/html", resp.ContentType)
	}
	if strings.TrimSpace(resp.Markdown) == "" {
		t.Fatalf("expected non-empty Markdown, got %q", resp.Markdown)
	}
	if !strings.Contains(resp.Markdown, "Hi") {
		t.Errorf("markdown missing heading text: %q", resp.Markdown)
	}
	if len(resp.Body) == 0 {
		t.Errorf("expected raw Body to be preserved")
	}
}

func TestGet_NonHTMLNotConverted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	resp, err := Get(context.Background(), srv.URL, Options{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Markdown != "" {
		t.Errorf("expected empty Markdown for non-HTML, got %q", resp.Markdown)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("Body = %q, want raw json", resp.Body)
	}
}
