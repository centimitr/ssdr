package main

import (
	"github.com/devbycm/scli"
	"github.com/devbycm/ssdr"
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	new(scli.App).
		Action(":service", registry).
		Run()
}
func registry(c *scli.Context) {
	r := ssdr.NewRegistry(c.Get("service"))
	log.Fatalln(r.Run())
}
