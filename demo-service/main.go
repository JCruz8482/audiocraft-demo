package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	gs "github.com/JCruz8482/audiocraft-demo/demo-service/gen_service"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const GEN_SERVICE_URL = "localhost:9000"

func main() {
	router := gin.Default()
	router.Static("/static", "./static")
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	router.GET("/progress", getAudioHandler)

	router.Run("localhost:8080")
}

func getAudioHandler(c *gin.Context) {
	prompt := c.Query("prompt")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	conn, err := grpc.Dial(GEN_SERVICE_URL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errMsg := "error dialing gen-service"
		log.Printf("%s%v", errMsg, err)
		c.IndentedJSON(http.StatusInternalServerError, errMsg)
		return
	}
	defer conn.Close()

	client := gs.NewAudioCraftGenServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	stream, err := client.GetAudioStream(ctx, &gs.GetAudioStreamRequest{Prompt: prompt})
	if err != nil {
		log.Printf("failed to call gen-service: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Printf("Failed to receive response stream data: %v", err)
			c.IndentedJSON(http.StatusInternalServerError, err)
			return
		}
		log.Printf("Received: %s", response.Progress)
		data := []byte("data: " + response.Progress + "\n\n")
		_, err = c.Writer.Write(data)
		if err != nil {
			log.Println("Error writing to client:", err)
			return
		}
		c.Writer.Flush()
		time.Sleep(10)
		if response.Progress == "Task completed" {
			data := []byte("data: Here is your data\n\n")
			_, err = c.Writer.Write(data)
			if err != nil {
				log.Println("Error writing to client:", err)
				return
			}
			c.Writer.Flush()
			return
		}
	}
}
