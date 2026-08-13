FROM golang:alpine AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -v -o /usr/local/bin/ktauth ./cmd/ktauth/main.go

FROM alpine:latest

COPY --from=builder /usr/local/bin/ktauth /ktauth

EXPOSE 51214

CMD [ "/ktauth" ]