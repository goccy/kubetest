FROM golang:1.16-stretch

ENV GOPATH /go
WORKDIR /go/src/github.com/goccy/kubetest

COPY ./go.* ./

RUN go mod download

COPY . .

RUN go build -o /go/bin/kubetest cmd/kubetest/main.go
