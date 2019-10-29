package util

import "github.com/sirupsen/logrus"

var Logger *logrus.Logger

func init(){
	Logger = logrus.New()
	Logger.Level = logrus.InfoLevel
}
