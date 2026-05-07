package fetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	resp, err := Get(context.Background(), srv.URL, Options{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(resp.Body) != "hello" {
		t.Fatalf("body = %q, want %q", resp.Body, "hello")
	}
	if resp.ContentType != "text/plain" {
		t.Fatalf("ContentType = %q, want %q", resp.ContentType, "text/plain")
	}
	if resp.Truncated {
		t.Fatalf("Truncated = true, want false")
	}
}

func TestGet_SizeCap(t *testing.T) {
	const max = int64(1024)
	big := strings.Repeat("a", int(10*max))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(big))
	}))
	defer srv.Close()

	resp, err := Get(context.Background(), srv.URL, Options{MaxBytes: max})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if int64(len(resp.Body)) != max {
		t.Fatalf("len(Body) = %d, want %d", len(resp.Body), max)
	}
	if !resp.Truncated {
		t.Fatalf("Truncated = false, want true")
	}
}

func TestGet_RedirectCap(t *testing.T) {
	const cap = 3
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var n int
		_, _ = fmt.Sscanf(r.URL.Query().Get("n"), "%d", &n)
		http.Redirect(w, r, fmt.Sprintf("%s/?n=%d", srv.URL, n+1), http.StatusFound)
	})

	_, err := Get(context.Background(), srv.URL+"/?n=0", Options{MaxRedirects: cap})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrTooManyRedirects) {
		t.Fatalf("err = %v, want errors.Is ErrTooManyRedirects", err)
	}
}

func TestGet_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer srv.Close()

	timeout := 50 * time.Millisecond
	start := time.Now()
	_, err := Get(context.Background(), srv.URL, Options{Timeout: timeout})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if elapsed > 2*timeout+200*time.Millisecond {
		t.Fatalf("Get took %v, want < %v", elapsed, 2*timeout)
	}
}
