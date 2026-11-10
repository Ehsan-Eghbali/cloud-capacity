FROM golang:1.27.3-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server main.go

FROM alpine:3.24

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server .
COPY config.yaml .
COPY migrations ./migrations
COPY seeds ./seeds

EXPOSE 8080

CMD ["./server", "server"]
