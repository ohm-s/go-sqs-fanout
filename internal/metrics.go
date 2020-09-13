package internal

import (
	"github.com/gin-gonic/gin"
	"github.com/ohm-s/go-sqs-fanout/pkg"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)


func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/health", healthHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return r
}

func healthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}


func SetupProm(httpCloseChannel <-chan int) chan int {
	var httpReadyChannel = make(chan int)
	log.Print(pkg.GetPromAddr())
	if pkg.GetPromAddr() != "" {

		gRouter := SetupRouter()
		httpServer := http.Server{
			Addr:    pkg.GetPromAddr(),
			Handler: gRouter,
		}
		go func() {
			go httpServer.ListenAndServe()
			close(httpReadyChannel)
			<-httpCloseChannel
			httpServer.Close()
		}()
	} else {
		close(httpReadyChannel)
	}
	return httpReadyChannel
}