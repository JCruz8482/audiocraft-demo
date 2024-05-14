# AudioCraft demo
## demo-service

#### Run
```
go run .
```

#### sqlc
sqlc generate

#### migrations
Must have golang-migrate installed

on mac:

```
brew install golange-migrate
```

create migration
```
migrate create -ext sql -dir ./db/migrations
```

run migrations
```
migrate -database 'postgres://jeremy@localhost:5432/suniduai?sslmode=disable' -path ./db/migrations up
```
