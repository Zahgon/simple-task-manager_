package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Log      LogConfig
}

type LogConfig struct {
	Level string `koanf:"level"`
}

type ServerConfig struct {
	Port            int           `koanf:"port"`
	ReadTimeout     time.Duration `koanf:"read_timeout"`
	WriteTimeout    time.Duration `koanf:"write_timeout"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host         string `koanf:"host"`
	Port         int    `koanf:"port"`
	User         string `koanf:"user"`
	Password     string `koanf:"password"`
	Name         string `koanf:"name"`
	SSLMode      string `koanf:"sslmode"`
	MigrationsPath string `koanf:"migrations_path"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

func Load() (Config, error) {
	k := koanf.New(".")

	defaults := map[string]interface{}{
		"server.port":             8080,
		"server.read_timeout":     10 * time.Second,
		"server.write_timeout":    10 * time.Second,
		"server.shutdown_timeout": 5 * time.Second,
		"db.host":                 "localhost",
		"db.port":                 5432,
		"db.user":                 "postgres",
		"db.password":             "postgres",
		"db.name":                 "tasks",
		"db.sslmode":              "disable",
		"db.migrations_path":      "file://migrations",
		"log.level":               "info",
	}

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return Config{}, fmt.Errorf("loading defaults: %w", err)
	}

	envProvider := env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	})
	if err := k.Load(envProvider, nil); err != nil {
		return Config{}, fmt.Errorf("loading env vars: %w", err)
	}

	var cfg Config
	if err := k.UnmarshalWithConf("server", &cfg.Server, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return Config{}, fmt.Errorf("unmarshalling server config: %w", err)
	}
	if err := k.UnmarshalWithConf("db", &cfg.Database, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return Config{}, fmt.Errorf("unmarshalling database config: %w", err)
	}
	if err := k.UnmarshalWithConf("log", &cfg.Log, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return Config{}, fmt.Errorf("unmarshalling log config: %w", err)
	}

	return cfg, nil
}
