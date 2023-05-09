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

func startGRPC(port string, mysql *database.Mysql) error {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	return server.NewGRPCServer(mysql).Serve(lis)
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

	// Connect to MySQL
	mysqlConStr := os.Getenv("MYSQL_CONNECTION_STRING")
	if len(mysqlConStr) == 0 {
		mysqlConStr = "root:docker@tcp(localhost:3306)/systera?charset=utf8mb4&parseTime=True&loc=Local"
	}
	mysqlClient := database.NewMysqlClient(mysqlConStr, "systera")

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

		if err := startGRPC(port, mysqlClient); err != nil {
			logrus.Fatalf("[GRPC] gRPC Error: %s", err)
		}
	}()
	<-wait
}
