package main

import (
	"fmt"
	"log"
	"net/http"
	"pagechecker/config"
	"pagechecker/internal/downloader"
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

		doc := downloader.Document{}
		if err := doc.DownloadDoc(c, req.URL); err != nil {
			views.RenderHtml(c, http.StatusBadGateway,
				views.ErrorPage(req.URL, err.Error()))
			return
		}

		// TODO: replace with real analysis of the submitted URL.
		result := views.Result{
			URL:         req.URL,
			HTMLVersion: "HTML5",
			Title:       "Example Domain",
			Headings:    views.HeadingCounts{1, 4, 7, 0, 0, 0},
			Links: views.LinkStats{
				Internal:     12,
				External:     5,
				Inaccessible: 2,
			},
			HasLoginForm: true,
		}
		views.RenderHtml(c, http.StatusOK, views.ResultPage(result))
	})

	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.ServerPort)); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
