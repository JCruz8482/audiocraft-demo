package main

import (
	"context"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	gs "github.com/JCruz8482/audiocraft-demo/demo-service/gen_service"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const GEN_SERVICE_URL = "gen-service:9000"

func main() {
	router := gin.Default()
	router.Static("/static", "./static")
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	router.GET("/progress", getAudioHandler)
	router.Run(":8080")
}

func streamAudioHandler(c *gin.Context) {
	file := "../soul.mp3"
	audioData, err := os.ReadFile(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read audio file")
		return
	}

	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	htmlResponse := "<audio controls><source src=\"data:audio/mpeg;base64," + audioBase64 + "\" type=\"audio/mpeg\"></audio>"
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, htmlResponse)
}

func streamAudio(path string, c *gin.Context) string {
	audioData, err := os.ReadFile(path)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read audio file")
		return "failed"
	}

	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	log.Println("audio base 64")
	log.Println(audioBase64)
	return audioBase64
}

func getAudioHandler(c *gin.Context) {
	prompt := c.Query("prompt")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()
	conn, err := grpc.Dial(GEN_SERVICE_URL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errMsg := "error dialing gen-service"
		log.Printf("%s%v", errMsg, err)
		c.IndentedJSON(http.StatusInternalServerError, errMsg)
		return
	}
	defer conn.Close()

	client := gs.NewAudioCraftGenServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
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
		str := response.Progress
		index := strings.Index(str, ":")

		if index != -1 {
			c.Writer.Write([]byte("data: DONE!\n\n"))
			path := str[index+1:]
			log.Println("path = " + path)
			path = strings.TrimSpace(path)
			path = "../" + path
			audio := streamAudio(path, c)
			log.Println(audio)
			data := []byte("data: audio: " + audio + "\n\n")
			_, err = c.Writer.Write(data)
			if err != nil {
				log.Println("Error writing to client:", err)
				return
			}

		}
		c.Writer.Flush()
		time.Sleep(10)
		if response.GetMessage() != "" {
			split := strings.Split(response.Message, ":")
			file := split[len(split)-1]
			streamAudio(file, c)
			c.Writer.Flush()
			return
		}
	}
}
