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
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const GEN_SERVICE_URL = "localhost:5000"
const AUDIO_BUCKET_NAME = "audiocraft-demo-bucket"
const AWS_REGION = "us-west-2"

type Task struct {
	Status Status
	Data   string
}

var (
	ERROR = errors.New("uh oh. failed to generate audio")

	aws_session *session.Session
	s3Client    *s3.S3

	taskMap = make(map[uuid.UUID]chan Task)
)

type Status string

const (
	Processing Status = "Processing"
	Done       Status = "Done"
	Error      Status = "Error"
)

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

	aws_session = session.Must(session.NewSession(&aws.Config{
		Region:           aws.String(AWS_REGION),
		Endpoint:         aws.String("http://localhost:9000"),
		S3ForcePathStyle: aws.Bool(true),
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		)}))
	s3Client = s3.New(aws_session)

	r := gin.Default()
	r.Use(gin.Logger())
	log.Println(os.Getenv("AWS_ACCESS_KEY_ID"))
	r.Static("/static", "./static")
	r.LoadHTMLGlob("views/*.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "gen", gin.H{})
	})
	r.GET("/login", func(c *gin.Context) {
		c.File("./static/login.html")
	})
	r.GET("/signup", func(c *gin.Context) {
		c.File("./static/signup.html")
	})
	r.POST("/login", loginHandler)
	r.POST("/signup", signUpHandler)
	r.GET("/hello", func(c *gin.Context) { c.IndentedJSON(http.StatusAccepted, "hello world") })
	r.POST("/generateAudio", postGenerateAudio)
	r.GET("/generateAudio/:id", getGenerateAudio)

	//r.Use(auth.AuthHandler)
	r.Run(":8080")
}

type LoginForm struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GenerateAudioReq struct {
	Prompt string `json:"prompt"`
}

func loginHandler(c *gin.Context) {
	var user LoginForm

	if err := c.BindJSON(&user); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	sessionKey, err := auth.Login(context.Background(), user.Email, user.Password)
	if err != nil {
		log.Println(err)
		c.IndentedJSON(http.StatusNotFound, "user not found")
		return
	}

	c.IndentedJSON(http.StatusOK, fmt.Sprintf(`{"sessionKey":"%s"}`, sessionKey))
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

func getPresignedUrl(objectKey string) (string, error) {
	req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(AUDIO_BUCKET_NAME),
		Key:    aws.String(objectKey),
	})
	url, err := req.Presign(15 * time.Minute)

	if err != nil {
		return "", err
	}

	log.Println("The URL is", url)
	return url, nil
}

type GenAudioResponse struct {
	ID     string `json:"id"`
	Data   string `json:"data"`
	Status Status `json:"status"`
	Error  string `json:"error"`
}

func postGenerateAudio(c *gin.Context) {
	var req GenerateAudioReq

	if err := c.BindJSON(&req); err != nil {
		c.Abort()
		return
	}

	prompt := req.Prompt
	id := uuid.New()

	taskChan := make(chan Task, 10)
	taskMap[id] = taskChan

	go generateAudio(id.String(), prompt, taskChan)

	c.IndentedJSON(200, GenAudioResponse{ID: id.String()})
}

func getGenerateAudio(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Charset", "utf-8")

	id := c.Param("id")

	err := uuid.Validate(id)
	if err != nil {
		c.JSON(404, GenAudioResponse{Error: "ID not found"})
		return
	}

	uuid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(404, GenAudioResponse{Error: "ID not found"})
		return
	}

	ch, ok := taskMap[uuid]
	if !ok {
		c.JSON(404, GenAudioResponse{Error: "ID not found"})
		return
	}

	for task := range ch {
		c.SSEvent("", GenAudioResponse{
			ID:     id,
			Status: task.Status,
			Data:   task.Data,
		})
		c.Writer.Flush()
	}
}

func generateAudio(id string, prompt string, taskChan chan Task) {
	defer close(taskChan)

	conn, err := grpc.Dial(GEN_SERVICE_URL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error dialing gen-service for task id: %s, err: %v", id, err)
		taskChan <- Task{Status: Error}
		return
	}
	defer conn.Close()

	client := gs.NewAudioCraftGenServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()

	stream, err := client.GetAudioStream(ctx, &gs.GetAudioStreamRequest{Prompt: prompt})

	if err != nil {
		log.Printf("failed to call gen-service for task id: %s, err: %v", id, err)
		taskChan <- Task{Status: Error}
		return
	}

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			log.Println("EOF reached in stream")
			return
		}

		if err != nil {
			log.Printf("Failed to receive response stream data: %v", err)
			/*
				c.SSEvent("", GenAudioResponse{
					ID:     id,
					Status: Error,
					Error:  "Failed to generate audio",
				})*/
			return
		}

		progress := response.Progress
		log.Println(progress)
		if strings.HasPrefix(progress, "object_key:") {
			s3Object := strings.SplitN(progress, ":", 2)[1]
			s3Object = strings.TrimSpace(s3Object)

			url, err := getPresignedUrl(s3Object)
			if err != nil {
				log.Printf("Error generating presigned url %v", err)
				/*
					c.SSEvent("", GenAudioResponse{
						ID:     id,
						Status: Error,
						Error:  "Failed to generate audio",
					})
				*/
				return
			}

			taskChan <- Task{
				Status: Done,
				Data:   url,
			}
			return
		}

		taskChan <- Task{Status: Processing}
	}
}
