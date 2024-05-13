-- name: CreateUser :one
INSERT INTO user_account (email, hashword)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserCredential :one
SELECT hashword
FROM user_account
WHERE email = $1;
