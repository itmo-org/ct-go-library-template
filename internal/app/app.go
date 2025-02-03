package app

import (
	"github.com/project/library/config"
	"go.uber.org/zap"
)

func Run(logger *zap.Logger, cfg *config.Config) {
	setupDatabaseMigrations(cfg, logger)
}

func runRest() {}

func runGrpc() {}
