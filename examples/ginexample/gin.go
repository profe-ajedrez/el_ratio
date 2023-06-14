package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/profe-ajedrez/el_ratio"
)

var l *el_ratio.LeakybuckerLimiter

func setupRouter() *gin.Engine {

	l = el_ratio.NewLeakyBucketLimiter(1, 5*time.Second)
	r := gin.New()
	r.Use(Limiter())

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	return r
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:3333
	r.Run(":3333")
}

func Limiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := l.Wait()

		fmt.Println(now)
		c.Next()
	}
}
