FROM golang:1.26.2-alpine3.23 AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -v -o /usr/local/bin/ktauth ./cmd/ktauth/main.go

FROM alpine:3.23

COPY --from=builder /usr/local/bin/ktauth /ktauth

EXPOSE 10000

CMD [ "/ktauth" ]