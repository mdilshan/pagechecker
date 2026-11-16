package linkchecker

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type LinkStatus struct {
	Url        string
	StatusCode int   // 0 when the request itself failed
	Err        error // network error, timeout, ...
}

func (s LinkStatus) Accessible() bool {
	return s.Err == nil && s.StatusCode < 400
}

var ErrUnsupportedScheme = errors.New("unsupported scheme, not checked")

var client = &http.Client{}

// per request total workers
var workers int

// total links check across entire server concurrently
var outbound chan struct{}

func Init(timeout time.Duration, perRequest, maxConcurrent int) {
	client.Timeout = timeout
	workers = perRequest
	outbound = make(chan struct{}, maxConcurrent)
}

func Check(ctx context.Context, links []string) []LinkStatus {
	jobs := make(chan string) // unbuffered, a send waits for a free worker
	results := make(chan LinkStatus, len(links))

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			worker(ctx, jobs, results)
		})
	}

	// results is buffered for every link, so recording here never blocks
	for _, link := range links {
		// mailto:, javascript:, tel: ... can't be checked over http
		if !strings.HasPrefix(link, "http://") &&
			!strings.HasPrefix(link, "https://") {
			results <- LinkStatus{Url: link, Err: ErrUnsupportedScheme}
			continue
		}

		select {
		case jobs <- link:
			slog.Debug("[LinkChcker] Queud work for", "link", link)
		case <-ctx.Done(): // user left, record it without checking
			results <- LinkStatus{Url: link, Err: ctx.Err()}
		}
	}
	close(jobs)
	wg.Wait()
	close(results)

	statuses := make([]LinkStatus, 0, len(results))
	for s := range results {
		if !s.Accessible() {
			slog.Warn("link not accessible",
				"url", s.Url, "status", s.StatusCode, "err", s.Err)
		}
		statuses = append(statuses, s)
	}

	return statuses
}

func worker(
	ctx context.Context, jobs <-chan string, results chan<- LinkStatus,
) {
	for url := range jobs {
		select {
		case outbound <- struct{}{}: // take a server wide slot
		case <-ctx.Done():
			results <- LinkStatus{Url: url, Err: ctx.Err()}
			continue
		}

		results <- check(ctx, url)
		<-outbound // give the slot back
	}
}

func check(ctx context.Context, url string) LinkStatus {
	status := LinkStatus{Url: url}

	code, err := request(ctx, http.MethodHead, url)
	if err == nil &&
		code == http.StatusMethodNotAllowed || code == http.StatusNotFound {
		// some servers don't allow HEAD so fallback to GET
		code, err = request(ctx, http.MethodGet, url)
	}

	status.StatusCode = code
	status.Err = err
	return status
}

func request(ctx context.Context, method, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close() // we only need the status code

	return resp.StatusCode, nil
}

// IsInternal reports if link points to the same host as the page.
func IsInternal(link string, base *url.URL) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, base.Host)
}
