package logger

import (
	"os"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// Init - InitLogging
func Init() {
	debug := os.Getenv("DEBUG")
	if len(debug) == 0 {
		// PRODUCTION
		logrus.SetFormatter(&prefixed.TextFormatter{
			TimestampFormat: "2006/01/02 15:04:05",
			FullTimestamp:   true,
		})
		logrus.SetOutput(os.Stdout)
	} else {
		// DEBUG
		logrus.SetFormatter(&prefixed.TextFormatter{
			TimestampFormat: "2006/01/02 15:04:05",
			FullTimestamp:   true,
		})
		logrus.SetOutput(os.Stdout)
		logrus.SetLevel(logrus.DebugLevel)
	}
}
