package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// StartLog is used to pretty print a log
// startup.
func StartLog(cmd string) {
	log.Info("----------------STARK----------------")
	log.Info("starting...")
	log.Info("\tcommand:\t", cmd)
	log.Info("\tconfig:\t", viper.ConfigFileUsed())
}