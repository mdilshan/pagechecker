//	go test ./internal/linkchecker/ -v

package linkchecker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	Init(10*time.Second, 10, 100)
	os.Exit(m.Run())
}

func newServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/missing", http.NotFound)
	mux.HandleFunc("/no-head", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// setLimits changes the worker count and the server wide cap for one
// test and puts the defaults back after it.
func setLimits(t *testing.T, w, cap int) {
	t.Helper()
	oldWorkers, oldOutbound := workers, outbound
	workers, outbound = w, make(chan struct{}, cap)
	t.Cleanup(func() { workers, outbound = oldWorkers, oldOutbound })
}

// slowServer counts how many requests it is handling at the same time
// and remembers the highest number in max.
func slowServer(t *testing.T, max *atomic.Int32) *httptest.Server {
	t.Helper()

	var running atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			n := running.Add(1)
			defer running.Add(-1)
			for {
				m := max.Load()
				if n <= m || max.CompareAndSwap(m, n) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
		}))
	t.Cleanup(srv.Close)
	return srv
}

func repeat(url string, n int) []string {
	links := make([]string, n)
	for i := range links {
		links[i] = url
	}
	return links
}

// go test ./internal/linkchecker/ -v -run TestCheck_Statuses
func TestCheck_Statuses(t *testing.T) {
	srv := newServer(t)

	links := []string{
		srv.URL + "/ok",
		srv.URL + "/missing",
		srv.URL + "/no-head",
		"http://127.0.0.1:1/down", // nothing listens on port 1
		"mailto:someone@example.com",
	}

	statuses := Check(context.Background(), links)

	// every link gets a status, mailto included
	if len(statuses) != len(links) {
		t.Fatalf("got %d statuses, want %d: %+v",
			len(statuses), len(links), statuses)
	}

	want := map[string]bool{
		srv.URL + "/ok":              true,
		srv.URL + "/missing":         false,
		srv.URL + "/no-head":         true,
		"http://127.0.0.1:1/down":    false,
		"mailto:someone@example.com": false,
	}
	for _, s := range statuses {
		if s.Accessible() != want[s.Url] {
			t.Errorf("%s accessible = %v, want %v (status=%d err=%v)",
				s.Url, s.Accessible(), want[s.Url], s.StatusCode, s.Err)
		}
	}
}

// go test ./internal/linkchecker/ -v -run TestCheck_WorkerLimit
func TestCheck_WorkerLimit(t *testing.T) {
	setLimits(t, 3, 100)

	var max atomic.Int32
	srv := slowServer(t, &max)

	statuses := Check(context.Background(), repeat(srv.URL, 30))

	if len(statuses) != 30 {
		t.Errorf("got %d statuses, want 30", len(statuses))
	}
	if got := max.Load(); got > 3 {
		t.Errorf("max concurrent requests = %d, want <= 3", got)
	}
}

// go test ./internal/linkchecker/ -v -run TestCheck_ServerWideLimit
func TestCheck_ServerWideLimit(t *testing.T) {
	// 4 requests with 3 workers each could run 12 at once,
	// the shared cap must keep it at 5
	setLimits(t, 3, 5)

	var max atomic.Int32
	srv := slowServer(t, &max)

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			Check(context.Background(), repeat(srv.URL, 10))
		})
	}
	wg.Wait()

	if got := max.Load(); got > 5 {
		t.Errorf("max concurrent requests = %d, want <= 5", got)
	}
}

// go test ./internal/linkchecker/ -v -run TestCheck_CanceledContext
func TestCheck_CanceledContext(t *testing.T) {
	srv := newServer(t)
	setLimits(t, 1, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan []LinkStatus)
	go func() {
		done <- Check(ctx, repeat(srv.URL+"/ok", 3))
	}()

	select {
	case statuses := <-done:
		if len(statuses) != 3 {
			t.Errorf("got %d statuses, want 3", len(statuses))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Check did not return after the context was canceled")
	}
}

// go test ./internal/linkchecker/ -v -run TestIsInternal
func TestIsInternal(t *testing.T) {
	base, _ := url.Parse("https://example.com/page")

	if !IsInternal("https://example.com/about", base) {
		t.Error("same host should be internal")
	}
	if IsInternal("https://other.com/", base) {
		t.Error("other host should be external")
	}
}
