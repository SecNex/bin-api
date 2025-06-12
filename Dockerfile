FROM golang:alpine AS builder

LABEL org.opencontainers.image.source=https://github.com/secnex/bin-api

WORKDIR /app

COPY go.mod ./

RUN go mod download && go mod verify

COPY . .

RUN go build -o bin-api main.go

FROM alpine:latest AS runner

WORKDIR /app

COPY --from=builder /app/bin-api .

EXPOSE 8081

CMD ["./bin-api"]