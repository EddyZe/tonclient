package config

import "github.com/sirupsen/logrus"

func InitLogger() *logrus.Logger {

	var logger = logrus.New()

	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
		ForceColors:   true,
	})

	return logger
}
