package auth

import (
	"os"
	"strconv"

	sesh "github.com/jcruz8482/oatAuth/session"
)

var SessionManager *sesh.SessionManager

func InitializeSessionManager() error {
	redisAddr := os.Getenv("REDIS_URL")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	expiration := os.Getenv("SESSION_EXPIRATION")

	exp, err := strconv.Atoi(expiration)
	if err != nil {
		return err
	}

	SessionManager, err = sesh.NewSessionManager(redisAddr, redisPassword, exp)
	if err != nil {
		return err
	}
	return nil
}
