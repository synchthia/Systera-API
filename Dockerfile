FROM golang:1.13.6 AS build
WORKDIR /go/src/github.com/synchthia/systera-api

ENV GOOS linux
ENV CGO_ENABLED 0

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -a -installsuffix cgo -v -o /systera cmd/systera/main.go

FROM alpine

RUN apk --no-cache add tzdata
COPY --from=build /systera /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/systera"]
