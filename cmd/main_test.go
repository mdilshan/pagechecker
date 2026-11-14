// Integration test: the real router, downloader, analyzer and link
// checker run together against a local fake website.
//
//	go test ./cmd/ -v

package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"pagechecker/config"
	"pagechecker/internal/downloader"
	"pagechecker/internal/linkchecker"
	"strings"
	"testing"
)

// same setup as main(), values come from config.ini
func TestMain(m *testing.M) {
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}
	downloader.Init(cfg.Downloader.Timeout, cfg.Downloader.MaxBodyMB)
	linkchecker.Init(cfg.LinkChecker.Timeout,
		cfg.LinkChecker.Workers, cfg.LinkChecker.MaxConcurrent)

	os.Exit(m.Run())
}

const fakePage = `<!DOCTYPE html>
<html><head><title>Fake Site</title></head>
<body>
	<h1>Welcome</h1>
	<a href="/ok">ok</a>
	<a href="/missing">missing</a>
	<form><input type="password"></form>
</body></html>`

func fakeSite(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fakePage))
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {})
	srv := httptest.NewServer(mux) // anything else is a 404
	t.Cleanup(srv.Close)
	return srv
}

func postAnalyze(siteurl string) *httptest.ResponseRecorder {
	form := url.Values{"siteurl": {siteurl}}
	req := httptest.NewRequest(http.MethodPost, "/analyze",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	setupRouter().ServeHTTP(w, req)
	return w
}

func TestAnalyze_Page(t *testing.T) {
	site := fakeSite(t)

	w := postAnalyze(site.URL + "/page")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"HTML5", "Fake Site", "badge-yes"} {
		if !strings.Contains(body, want) {
			t.Errorf("result page is missing %q", want)
		}
	}
}

func TestAnalyze_PageNotFound(t *testing.T) {
	site := fakeSite(t)

	w := postAnalyze(site.URL + "/nope")

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", w.Code)
	}
	if !strings.Contains(w.Body.String(), "404 Not Found") {
		t.Error("error page should show the 404")
	}
}

func TestAnalyze_InvalidURL(t *testing.T) {
	w := postAnalyze("not a url")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
