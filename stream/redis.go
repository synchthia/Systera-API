package stream

import (
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/sirupsen/logrus"
)

var pool *redis.Pool

// NewRedisPool - redis Connection Pooling
func NewRedisPool(server string) {
	logrus.WithFields(logrus.Fields{
		"server": server,
	}).Infof("[Redis] Creating Pool...")

	pool = &redis.Pool{
		MaxIdle:   12,
		MaxActive: 0,
		//IdleTimeout: 240 * time.Second,
		Wait: true,

		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)

			if err != nil {
				logrus.WithError(err).Errorf("[Redis] Error occurred in Connecting: %s", server)
				return nil, err
			}

			return c, err
		},

		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if err != nil {
				logrus.WithError(err).Errorf("[Redis] Error occurred in Redis Pool: %s", server)
			}
			return err
		},
	}
}
