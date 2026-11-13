package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"pagechecker/config"
	"pagechecker/internal/analyzer"
	"pagechecker/internal/downloader"
	"pagechecker/internal/linkchecker"
	"pagechecker/static"
	"pagechecker/views"

	"github.com/gin-gonic/gin"
)

type analyzeRequest struct {
	URL string `form:"siteurl" binding:"required,url"`
}

func main() {
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}

	r := setupRouter()

	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.ServerPort)); err != nil {
		log.Fatalf("failed to run server: %v", err)
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
			c.String(http.StatusBadRequest, "invalid url")
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
			views.RenderHtml(c, http.StatusBadGateway,
				views.ErrorPage(req.URL, err.Error()))
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
