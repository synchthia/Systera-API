package main

import (
	"io"
	"log"
	"net"
	"os"

	"gitlab.com/Startail/Systera-API/database"
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
	// Enable Logging to file
	logfile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic("Could not open server.log:" + err.Error())
	}
	defer logfile.Close()
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	log.SetFlags(log.Ldate | log.Ltime)

	// Init
	log.Printf("[API]: Starting SYSTERA-API Server...")

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

		log.Printf("[GRPC]: Listening %s", port)
		if err := startGRPC(port); err != nil {
			log.Fatalf("[!!!]: gRPC ERROR: %s", err)
		}
	}()
	<-wait
}
