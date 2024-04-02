package logging

import (
	"github.com/abiondevelopment/external-dns-webhook-abion/webhook/configuration"
	log "github.com/sirupsen/logrus"
)

func Init(config *configuration.Configuration) {
	setLogLevel(config.Debug)
	setLogFormat(config.LogFormat)
}

func setLogLevel(debugEnabled bool) {
	var logLevel log.Level
	if debugEnabled {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)
}

func setLogFormat(logFormat string) {
	if logFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{})
	}
}
