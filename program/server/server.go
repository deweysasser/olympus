package server

import (
	"fmt"
	"github.com/deweysasser/olympus/middleware"
	"github.com/gin-gonic/gin"
)

type Options struct {
	Port int `help:"Port on which to listen" default:"8081"`
	// DataDirectory string `help:"Directory into which to write data" type:"existingdir" default:"received"`
}

func (o *Options) Run() error {

	r := o.createServer()
	return r.Run(fmt.Sprintf(":%d", o.Port))
}

func (o *Options) createServer() *gin.Engine {
	r := gin.New()
	// r.Use(ginzerolog.Logger("gin"))
	r.Use(middleware.GinRequestLogger())
	r.GET("/status", func(context *gin.Context) {
		context.JSON(200, gin.H{"status": "alive"})
	})

	return r
}
