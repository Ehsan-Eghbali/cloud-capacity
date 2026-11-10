.PHONY: build run fmt vet migrate-up migrate-down seed worker up down logs clean

build:
	go build -o bin/server main.go

run:
	go run main.go server -c config.yaml

fmt:
	gofmt -l -w .

vet:
	go vet ./...

migrate-up:
	go run main.go migrate up -c config.yaml

migrate-down:
	go run main.go migrate down -c config.yaml

seed:
	go run main.go seed -c config.yaml

worker:
	go run main.go worker -c config.yaml

up:
	docker compose up --build -d

down:
	docker compose down -v

logs:
	docker compose logs -f app

clean:
	rm -rf bin
