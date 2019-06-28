package main

import (
	"net"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/synchthia/systera-api/database"
	"github.com/synchthia/systera-api/logger"
	"github.com/synchthia/systera-api/server"
	"github.com/synchthia/systera-api/stream"
)

func startGRPC(port string) error {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	return server.NewGRPCServer().Serve(lis)
}

func main() {
	// Init Logger
	logger.Init()

	// Init
	logrus.Printf("[API] Starting SYSTERA-API Server...")

	// Redis
	go func() {
		redisAddr := os.Getenv("REDIS_ADDRESS")
		if len(redisAddr) == 0 {
			redisAddr = "localhost:6379"
		}
		stream.NewRedisPool(redisAddr)
	}()

	// MongoDB
	mongoAddr := os.Getenv("SYSTERA_MONGO_ADDRESS")
	if len(mongoAddr) == 0 {
		mongoAddr = "localhost:27017"
	}
	database.NewMongoSession(mongoAddr)

	// gRPC
	wait := make(chan struct{})
	go func() {
		defer close(wait)
		port := os.Getenv("GRPC_LISTEN_PORT")
		if len(port) == 0 {
			port = ":17300"
		}

		msg := logrus.WithField("listen", port)
		msg.Infof("[GRPC] Listening %s", port)

		if err := startGRPC(port); err != nil {
			logrus.Fatalf("[GRPC] gRPC Error: %s", err)
		}
	}()
	<-wait
}
