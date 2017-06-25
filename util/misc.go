package util

import "github.com/sirupsen/logrus"

// HandleError : General Error Handling
func HandleError(err error) error {
	if err != nil {
		logrus.Warnf("[!!!] An expected Error occurred! %s", err)
		return err
	}
	return nil
}
