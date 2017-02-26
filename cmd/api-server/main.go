package main

import (
	"io"
	"log"
	"os"
)

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
}
