FROM golang:1.14-stretch AS builder

ENV GOPATH /go
WORKDIR /go/src/github.com/goccy/kubetest

COPY ./go.* ./

RUN go mod download

COPY . .

RUN go build -o kubetest cmd/kubetest/main.go

FROM golang:1.14-stretch

ENV GOPATH /go

COPY --from=builder /go/src/github.com/goccy/kubetest/kubetest /go/bin/kubetest
