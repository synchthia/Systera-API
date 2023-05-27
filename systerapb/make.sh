#!/bin/bash

cd $(dirname $0)
BASEMENT=$PWD

export PATH="$PATH:$(go env GOPATH)/bin"
# https://grpc.io/docs/languages/go/quickstart/
go install -v google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

echo "Generating Go Protoc..."
#protoc --go_out=plugins=grpc:. *.proto
protoc -I . \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=require_unimplemented_servers=false \
    *.proto
