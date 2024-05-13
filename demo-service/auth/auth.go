package auth

import (
	"log"

	"github.com/JCruz8482/audiocraft-demo/demo-service/db"
	"github.com/jcruz8482/oatAuth"
	"golang.org/x/net/context"
)

func Login(
	ctx context.Context,
	email string,
	password string,
) (bool, error) {
	// if email not in db, return no user
	// if hashwords aren't equal return no user else return session token
	hashword, err := db.DB.GetUserCredential(ctx, email)
	if err != nil {
		return false, err
	}
	isMatch, err := argon2id.ComparePasswordAndHash(password, hashword)
	if err != nil {
		return false, err
	}
	return isMatch, nil
}

func SignUp(
	ctx context.Context,
	email string,
	password string,
) (bool, error) {
	hashword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Println("Aw shit we failed to create the hash")
		return false, err
	}

	_, err = db.DB.CreateUser(ctx, db.CreateUserParams{
		Email:    email,
		Hashword: hashword,
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}
