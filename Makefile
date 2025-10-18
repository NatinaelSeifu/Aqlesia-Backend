run:
	go run cmd/main.go
sqlc:
	~/go/bin/sqlc generate -f config/sqlc.yaml
migrate-down:
	- migrate -database postgresql://qetero:qeteroo123@localhost:5432/aqlesia_test?sslmode=disable -path ./internal/constants/query/schemas -verbose down
migrate-up:
	- migrate -database postgresql://qetero:qeteroo123@localhost:5432/aqlesia_test?sslmode=disable -path ./internal/constants/query/schemas -verbose up
migrate-create:
	- migrate create -ext sql -dir internal/constants/query/schemas -tz "UTC" $(ARGS)
go-test:
	go test ./... -p=1 -count=1 
swagger-gen:
	-~/go/bin/swag init -g initiator/initiator.go

