package main

import (
	"github.com/project/library/config"
	"github.com/project/library/internal/app"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.NewConfig()

	if err != nil {
		log.Fatalf("can not get application config: %s", err)
	}

	var logger *zap.Logger

	logger, err = zap.NewProduction()

	if err != nil {
		log.Fatalf("can not initialize logger: %s", err)
	}

	app.Run(logger, cfg)
}
