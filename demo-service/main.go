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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const GEN_SERVICE_URL = "localhost:5000"
const AUDIO_BUCKET_NAME = "audiocraft-demo-bucket"
const AWS_REGION = "us-west-2"

var aws_session = session.Must(session.NewSession(&aws.Config{
	Region:           aws.String(AWS_REGION),
	Endpoint:         aws.String("http://localhost:9000"),
	S3ForcePathStyle: aws.Bool(true),
	Credentials: credentials.NewStaticCredentials(
		"minioadmin",
		"minioadmin",
		"",
	),
}))

func main() {
	router := gin.Default()
	router.Static("/static", "./static")
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	router.GET("/progress", getAudioHandler)
	router.Run(":8080")
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

func downloadS3(objectKey string) string {
	file, err := os.Create(objectKey)
	if err != nil {
		log.Printf("Unable to open file %q, %v", objectKey, err)
		return ""
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(aws_session)
	log.Println(objectKey)
	log.Println(AUDIO_BUCKET_NAME)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(AUDIO_BUCKET_NAME),
			Key:    aws.String(objectKey),
		})
	if err != nil {
		log.Printf("failed to download: %v", err)
		return ""
	}
	return file.Name()
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
		progress := response.Progress
		log.Printf("Received: %s", progress)
		if strings.HasPrefix(progress, "object_key:") {
			s3Object := strings.SplitN(progress, ":", 2)[1]
			s3Object = strings.TrimSpace(s3Object)
			filename := downloadS3(s3Object)
			audio := streamAudio(filename, c)
			data := []byte("data: audio: " + audio + "\n\n")
			_, err = c.Writer.Write(data)
			if err != nil {
				log.Printf("failed to write audio data to stream: %v", err)
			}
			c.Writer.Flush()
			return
		}

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
