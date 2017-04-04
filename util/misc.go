package util

import "github.com/labstack/gommon/log"

func HandleError(err error) error {
	if err != nil {
		log.Printf("[!!!]: An expected Error occurred!", err)
		return err
	}
	return nil
}
