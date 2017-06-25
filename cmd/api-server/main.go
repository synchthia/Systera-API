package main

import (
	"net"
	"os"

	"github.com/sirupsen/logrus"

	"gitlab.com/Startail/Systera-API/database"
	"gitlab.com/Startail/Systera-API/logger"
	"gitlab.com/Startail/Systera-API/server"
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

	// MongoDB
	mongoAddr := os.Getenv("SYSTERA_MONGO_ADDRESS")
	if len(mongoAddr) == 0 {
		mongoAddr = "192.168.99.100:27017"
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
