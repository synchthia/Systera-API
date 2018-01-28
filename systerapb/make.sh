#!/bin/bash

echo "Generating Go Protoc..."
protoc --go_out=plugins=grpc:. *.proto

echo "Generating Java Protoc..."
protoc --java_out=. --plugin=protoc-gen-grpc-java=$HOME/bin/protoc-gen-grpc-java --grpc-java_out=. *.proto