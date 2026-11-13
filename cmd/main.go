package main

import (
	"fmt"
	"log"
	"net/http"
	"pagechecker/analyzer"
	"pagechecker/config"
	"pagechecker/static"
	"pagechecker/views"

	"github.com/gin-gonic/gin"
)

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
		// TODO: replace with real analysis of the submitted URL.
		result := analyzer.Result{
			URL:         c.PostForm("siteurl"),
			HTMLVersion: "HTML5",
			Title:       "Example Domain",
			Headings:    analyzer.HeadingCounts{1, 4, 7, 0, 0, 0},
			Links: analyzer.LinkStats{
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
