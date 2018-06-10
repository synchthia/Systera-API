#!/bin/bash

cd $(dirname $0)
BASEMENT=$PWD

PATH=$PATH:$GOPATH/bin
go get -u -v github.com/golang/protobuf/protoc-gen-go

echo "Generating Go Protoc..."
protoc --go_out=plugins=grpc:. *.proto