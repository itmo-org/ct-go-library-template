package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/mattes/migrate/source/file"
	"github.com/project/library/config"
	"go.uber.org/zap"
)

const (
	_defaultAttempts = 5
	_defaultTimeout  = time.Second
)

// NOTE: A simplified approach to avoid running migration containers.
func setupDatabaseMigrations(cfg *config.Config, logger *zap.Logger) {
	var (
		attempts = _defaultAttempts
		err      error
		m        *migrate.Migrate
	)

	wd, err := os.Getwd()

	if err != nil {
		logger.Error("can not get working directory", zap.Error(err))
		os.Exit(-1)
	}

	p, err := resolveMigrationDirectory(wd, "migrations")

	if err != nil {
		logger.Error("can not resolve migrations directory", zap.Error(err))
		os.Exit(-1)
	}

	for attempts > 0 {
		m, err = migrate.New("file://"+p, cfg.PG.URL)
		logger.Info("miration status", zap.Error(err))

		if err == nil {
			break
		}

		logger.Info("postgres is trying to connect, attempts left", zap.Int("attempts", attempts))

		time.Sleep(_defaultTimeout)
		attempts--
	}

	if err != nil {
		logger.Error("postgres connect error", zap.Error(err))
		os.Exit(-1)
	}

	defer func() {
		err, dbErr := m.Close()

		if err != nil || dbErr != nil {
			logger.Error("got migration close error", zap.Any("err", err), zap.Any("dbErr", dbErr))
		}
	}()

	err = m.Up()

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("migration up error", zap.Error(err))
		os.Exit(-1)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("no migration changes")
		return
	}

	logger.Info("migration up success")
}

func resolveMigrationDirectory(root string, directoryName string) (string, error) {
	cleanedRoot := filepath.Clean(root)
	var result string

outer:
	for {
		entries, err := os.ReadDir(cleanedRoot)

		if err != nil {
			return "", fmt.Errorf("can not read dir entires of root: %s", root)
		}

		for _, e := range entries {
			if !e.IsDir() && e.Name() == "go.mod" {
				break outer
			}
		}

		if cleanedRoot == "" {
			return "", fmt.Errorf("mark go.mod not found")
		}

		cleanedRoot = filepath.Dir(cleanedRoot)
	}

	err := filepath.WalkDir(cleanedRoot, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		if name == directoryName {
			result = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("walk fail tree fail, error: %w", err)
	}

	if result == "" {
		return "", fmt.Errorf("directory %s not found in root %s", directoryName, root)
	}

	return result, nil
}
