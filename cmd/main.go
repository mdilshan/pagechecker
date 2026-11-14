package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"pagechecker/config"
	"pagechecker/internal/analyzer"
	"pagechecker/internal/downloader"
	"pagechecker/internal/linkchecker"
	"pagechecker/static"
	"pagechecker/views"
	"syscall"

	"github.com/gin-gonic/gin"
)

type analyzeRequest struct {
	URL string `form:"siteurl" binding:"required,url"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load("")
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	downloader.Init(cfg.Downloader.Timeout, cfg.Downloader.MaxBodyMB)
	linkchecker.Init(cfg.LinkChecker.Timeout,
		cfg.LinkChecker.Workers, cfg.LinkChecker.MaxConcurrent)

	r := setupRouter()

	slog.Info("server starting", "port", cfg.Server.ServerPort)
	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.ServerPort)); err != nil {
		slog.Error("failed to run server", "err", err)
		os.Exit(1)
	}
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.StaticFS("/static", http.FS(static.FS))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/", func(c *gin.Context) {
		views.RenderHtml(c, http.StatusOK, views.IndexPage())
	})

	r.POST("/analyze", func(c *gin.Context) {
		var req analyzeRequest
		if err := c.ShouldBind(&req); err != nil {
			views.RenderHtml(c, http.StatusBadRequest,
				views.ErrorPage(req.URL, 0,
					"Please enter a full url, like https://example.com", ""))
			return
		}

		// the body is streamed straight into the analyzer, never kept
		// in memory as a whole
		var base *url.URL
		var data analyzer.PageData
		doc := downloader.Document{}
		err := doc.DownloadDoc(c, req.URL, func(r io.Reader) error {
			var err error
			base, err = url.Parse(doc.Url)
			if err != nil {
				return err
			}

			data, err = analyzer.Analyze(r, base)
			return err
		})
		if err != nil {
			status := 0
			if doc.Status >= 400 {
				status = doc.Status
			}
			views.RenderHtml(c, http.StatusBadGateway,
				views.ErrorPage(req.URL, status,
					errorMessage(status, err), err.Error()))
			return
		}

		statuses := linkchecker.Check(c, data.Links)

		links := views.LinkStats{Inaccessible: len(data.MalformedLinks)}
		for _, s := range statuses {
			if linkchecker.IsInternal(s.Url, base) {
				links.Internal++
			} else {
				links.External++
			}
			if !s.Accessible() {
				links.Inaccessible++
			}
		}

		var headings views.HeadingCounts
		for i := range headings {
			headings[i] = data.Headings[fmt.Sprintf("h%d", i+1)]
		}

		result := views.Result{
			URL:          req.URL,
			HTMLVersion:  data.HTMLVersion,
			Title:        data.Title,
			Links:        links,
			Headings:     headings,
			HasLoginForm: data.HasLoginForm,
		}
		views.RenderHtml(c, http.StatusOK, views.ResultPage(result))
	})

	return r
}

func errorMessage(status int, err error) string {
	var dnsErr *net.DNSError
	var netErr net.Error

	switch {
	case status == http.StatusNotFound:
		return "The page does not exist on that site."
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return "The site does not allow access to this page."
	case status >= 500:
		return "The site has a problem on its side, try again later."
	case status >= 400:
		return "The site refused the request."
	case errors.Is(err, downloader.ErrNotHTML):
		return "The url does not point to an html page."
	case errors.As(err, &dnsErr):
		return "The host could not be found, check the url."
	case errors.As(err, &netErr) && netErr.Timeout():
		return "The site took too long to respond."
	case errors.Is(err, syscall.ECONNREFUSED):
		return "The site refused the connection."
	default:
		return "The site could not be reached."
	}
}
