package main

import (
	"github.com/cloudimpl/next-coder-sdk/polycode"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "polycode/agent-app/.polycode"
	"polycode/agent-app/lib"
)

func main() {
	v := lib.NewValidator()
	polycode.SetValidator(v)

	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "x-polycode-partition-key")
	r.Use(cors.New(config))

	polycode.StartApp(r)
}
