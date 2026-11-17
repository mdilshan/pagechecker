//	go test ./internal/downloader/ -v

package downloader

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	Init(10*time.Second, 5)
	os.Exit(m.Run())
}

const page = "<!DOCTYPE html><html><head><title>Test</title></head>" +
	"<body><h1>Hi</h1></body></html>"

var longPage = page + strings.Repeat("<p>x</p>", 200)

func newServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func noContentType(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header()["Content-Type"] = nil
		w.Write([]byte(body))
	}
}

func withContentType(contentType, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte(body))
	}
}

// readInto returns a read callback that stores the body in got.
func readInto(got *string) func(io.Reader) error {
	return func(r io.Reader) error {
		b, err := io.ReadAll(r)
		*got = string(b)
		return err
	}
}

// discard is a read callback that ignores the body.
func discard(r io.Reader) error {
	_, err := io.Copy(io.Discard, r)
	return err
}

// go test ./internal/downloader/ -v -run TestDownloadDoc_Success
func TestDownloadDoc_Success(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		body    string
	}{
		{"plain content type", withContentType("text/html", page), page},
		{"content type with charset",
			withContentType("text/html; charset=utf-8", page), page},
		{"sniffed html", noContentType(page), page},
		{"sniffed html longer than 512 bytes",
			noContentType(longPage), longPage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newServer(t, tt.handler)

			var d Document
			var got string
			err := d.DownloadDoc(context.Background(), srv.URL,
				readInto(&got))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Status != http.StatusOK {
				t.Errorf("Status = %d, want %d", d.Status, http.StatusOK)
			}
			if got != tt.body {
				t.Errorf("Body = %q, want %q", got, tt.body)
			}
			if d.Url != srv.URL {
				t.Errorf("Url = %q, want %q", d.Url, srv.URL)
			}
		})
	}
}

func TestDownloadDoc_Errors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{"not found", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope", http.StatusNotFound)
		}, "status 404"},
		{"server error", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}, "status 500"},
		{"json content type",
			withContentType("application/json", `{"a":1}`), "not HTML"},
		{"sniffed plain text", noContentType("just some text"), "not HTML"},
		{"empty body without content type",
			noContentType(""), "html detection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newServer(t, tt.handler)

			called := false
			read := func(r io.Reader) error { called = true; return nil }

			var d Document
			err := d.DownloadDoc(context.Background(), srv.URL, read)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if called {
				t.Error("read was called for a failed download")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// go test ./internal/downloader/ -v -run TestDownloadDoc_SizeLimit
func TestDownloadDoc_SizeLimit(t *testing.T) {
	big := "<html><body>" + strings.Repeat("a", maxBodySize) +
		"</body></html>"
	srv := newServer(t, withContentType("text/html", big))

	var d Document
	var got string
	err := d.DownloadDoc(context.Background(), srv.URL, readInto(&got))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != maxBodySize {
		t.Errorf("read %d bytes, want %d", len(got), maxBodySize)
	}
}

// go test ./internal/downloader/ -v -run TestDownloadDoc_ReadError
func TestDownloadDoc_ReadError(t *testing.T) {
	srv := newServer(t, withContentType("text/html", page))
	readErr := errors.New("tokenizer failed")

	var d Document
	err := d.DownloadDoc(context.Background(), srv.URL,
		func(r io.Reader) error { return readErr })
	if !errors.Is(err, readErr) {
		t.Errorf("error = %v, want %v", err, readErr)
	}
}

func TestDownloadDoc_FollowsRedirect(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/new", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/new", withContentType("text/html", page))
	srv := newServer(t, mux.ServeHTTP)

	var d Document
	err := d.DownloadDoc(context.Background(), srv.URL+"/old", discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := srv.URL + "/new"; d.Url != want {
		t.Errorf("Url = %q, want %q", d.Url, want)
	}
}

func TestDownloadDoc_SendsAcceptHeader(t *testing.T) {
	var got string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Accept")
		withContentType("text/html", page)(w, r)
	})

	var d Document
	err := d.DownloadDoc(context.Background(), srv.URL, discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ACCEPT {
		t.Errorf("Accept = %q, want %q", got, ACCEPT)
	}
}

func TestDownloadDoc_InvalidURL(t *testing.T) {
	var d Document
	err := d.DownloadDoc(context.Background(), "://bad-url", discard)
	if err == nil || !strings.Contains(err.Error(), "build request") {
		t.Errorf("error = %v, want a build request error", err)
	}
}

func TestDownloadDoc_CanceledContext(t *testing.T) {
	srv := newServer(t, withContentType("text/html", page))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var d Document
	err := d.DownloadDoc(ctx, srv.URL, discard)
	if err == nil || !strings.Contains(err.Error(), "request") {
		t.Errorf("error = %v, want a request error", err)
	}
}
