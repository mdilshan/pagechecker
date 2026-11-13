package main

import (
	"fmt"
	"log"
	"net/http"
	"pagechecker/config"
	"pagechecker/views"

	_ "pagechecker/views"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/", func(c *gin.Context) {
		views.RenderHtml(c, http.StatusOK, views.Layout())
	})

	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.ServerPort)); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
