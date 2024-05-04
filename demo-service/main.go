package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

const GEN_SERVICE_URL = "localhost:9000"

func main() {
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, "hello world")
	})
	router.GET("/audio", getAudioHandler)

	router.Run("localhost:8080")
}

func getAudioHandler(c *gin.Context) {
	res, err := http.Get(GEN_SERVICE_URL)
	if err != nil {
		fmt.Println("ah fuck")
		c.IndentedJSON(http.StatusInternalServerError, "fucked up calling gen-service")
	}
	c.IndentedJSON(http.StatusOK, res)
}
