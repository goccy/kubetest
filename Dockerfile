FROM golang:1.22.0-bookworm

ENV GOPATH /go
WORKDIR /go/src/github.com/goccy/kubetest

COPY ./go.* ./

RUN go mod download

COPY . .

RUN go build -o /go/bin/kubetest cmd/kubetest/main.go
RUN go build -o /go/bin/kubetest-agent cmd/kubetest-agent/main.go

FROM golang:1.22.0-bookworm AS agent

COPY --from=0 /go/bin/kubetest-agent /bin/kubetest-agent
