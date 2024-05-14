package auth

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/JCruz8482/audiocraft-demo/demo-service/db"
	"github.com/gin-gonic/gin"
	"github.com/jcruz8482/oatAuth/argon2id"
)

func AuthHandler(c *gin.Context) {
	bearerAuth := c.GetHeader("Authorization")
	log.Print("headder = ")
	log.Println(bearerAuth)
	token := strings.Split(bearerAuth, " ")
	log.Print("token from header: ")
	log.Println(token)
	log.Println("indexed")
	log.Print(token[1])
	id, err := SessionManager.UserIdForSession(context.Background(), token[1])
	if err != nil {
		c.JSON(401, "no user logged in")
		// redirect to login page
		log.Println("error getting id for session")
		return
	}

	log.Print("user id = ")
	log.Println(id)
	c.Set("userId", id)
	c.Next()
}

func Login(
	ctx context.Context,
	email string,
	password string,
) (string, error) {
	// if email not in db, return no user
	// if hashwords aren't equal return no user else return session token
	account, err := db.DB.GetUserAccount(ctx, email)
	if err != nil {
		return "", err
	}
	isMatch, err := argon2id.ComparePasswordAndHash(password, account.Hashword)
	if err != nil {
		return "", err
	}
	if !isMatch {
		return "", errors.New("Passwords did not match")
	}

	id := account.ID.Bytes
	token, err := SessionManager.NewSession(ctx, string(id[:]))
	if err != nil {
		return "", err
	}
	return token, nil
}

func SignUp(
	ctx context.Context,
	email string,
	password string,
) (*db.UserAccount, error) {
	hashword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return &db.UserAccount{}, err
	}

	user, err := db.DB.CreateUser(ctx, db.CreateUserParams{
		Email:    email,
		Hashword: hashword,
	})
	if err != nil {
		log.Printf("error creating user %v", err)
		return &db.UserAccount{}, err
	}
	return &user, nil
}
