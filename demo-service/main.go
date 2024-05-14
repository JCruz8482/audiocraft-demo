package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/JCruz8482/audiocraft-demo/demo-service/auth"
	"github.com/JCruz8482/audiocraft-demo/demo-service/db"
	gs "github.com/JCruz8482/audiocraft-demo/demo-service/gen_service"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const GEN_SERVICE_URL = "localhost:5000"
const AUDIO_BUCKET_NAME = "audiocraft-demo-bucket"
const AWS_REGION = "us-west-2"

var ERROR = errors.New("uh oh. failed to generate audio")

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
	godotenv.Load()
	ctx := context.Background()
	conn, err := db.InitializeDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	log.Print("Initialized DB")
	err = auth.InitializeSessionManager()
	if err != nil {
		log.Fatal(err)
	}
	defer auth.SessionManager.Close()

	r := gin.Default()
	r.Use(gin.Logger())

	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	r.GET("/login", func(c *gin.Context) {
		c.File("./static/login.html")
	})
	r.GET("/signup", func(c *gin.Context) {
		c.File("./static/signup.html")
	})
	r.POST("/login", loginHandler)
	r.POST("/signup", signUpHandler)
	r.Use(auth.AuthHandler)
	r.GET("/hello", func(c *gin.Context) { c.IndentedJSON(http.StatusAccepted, "hello world") })
	r.GET("/generateAudio", generateAudio)
	r.Run(":8080")
}

type LoginForm struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func loginHandler(c *gin.Context) {
	var user LoginForm

	if err := c.BindJSON(&user); err != nil {
		log.Println("aw shit")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	token, err := auth.Login(context.Background(), user.Email, user.Password)
	if err != nil {
		log.Println("fuck")
		log.Println(err)
		c.IndentedJSON(http.StatusNotFound, "user not found")
		return
	}
	jsonn := fmt.Sprintf(`{"sessionKey":"%s"}`, token)
	log.Println(jsonn)
	log.Println("returning")
	c.IndentedJSON(http.StatusOK, jsonn)
}

func signUpHandler(c *gin.Context) {
	var user LoginForm
	if err := c.BindJSON(&user); err != nil {
		fmt.Println(err)
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	userAccount, err := auth.SignUp(context.Background(), user.Email, user.Password)
	if err != nil {

		fmt.Println(err)
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	c.IndentedJSON(http.StatusOK, userAccount)
}

func encodeAudio(path string) (string, error) {
	audioData, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read audio file %v", err)
		return "", ERROR
	}
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	return audioBase64, nil
}

func downloadS3(objectKey string) (string, error) {
	file, err := os.Create(objectKey)
	if err != nil {
		log.Printf("Unable to open file %q, %v", objectKey, err)
		return "", ERROR
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
		return "", ERROR
	}
	return file.Name(), nil
}

func generateAudio(c *gin.Context) {
	prompt := c.Query("prompt")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	conn, err := grpc.Dial(GEN_SERVICE_URL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errMsg := "error dialing gen-service"
		log.Printf("%s%v", errMsg, err)
		c.Header("Content-Type", "text/event-stream")
		c.IndentedJSON(http.StatusInternalServerError, errMsg)
		c.Writer.Flush()
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
			return
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
			filename, err := downloadS3(s3Object)
			if err != nil {
				c.Request.Close = true
			}
			audio, err := encodeAudio(filename)
			data := []byte("data: { \"audio\": \"" + audio + "\"}\n\n")
			_, err = c.Writer.Write(data)
			if err != nil {
				log.Printf("failed to write audio data to stream: %v", err)
				return
			}
			c.Writer.Flush()
			return
		}

		if response.Progress == "\"" {
			progress = "\\"
		} else {
			progress = response.Progress
		}
		data := []byte(fmt.Sprintf("data: { \"progress\": \"%s\" }\n\n", progress))
		_, err = c.Writer.Write(data)
		if err != nil {
			log.Println("Error writing to client:", err)
			return
		}
		c.Writer.Flush()
	}
}
