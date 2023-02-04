FROM golang:1.20 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /go/bin/app

ENTRYPOINT ["/go/bin/app"]
