package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr     string `yaml:"addr"`
		LogLevel string `yaml:"logLevel"`
	} `yaml:"server"`

	Postgres struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		DBName   string `yaml:"dbname"`
		MaxConns int    `yaml:"maxConns"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"postgres"`

	Listener struct {
		Channel          string `yaml:"channel"`
		ReconnectSeconds int    `yaml:"reconnectSeconds"`
	} `yaml:"listener"`
}

func Load() Config {
	env := strings.ToLower(os.Getenv("ENV"))
	if env == "" {
		env = "dev"
	}

	basePath := "configs"

	cfg := Config{}
	loadYAML(filepath.Join(basePath, "application.yaml"), &cfg)

	envFile := filepath.Join(basePath, env+".yaml")
	if _, err := os.Stat(envFile); err == nil {
		loadYAML(envFile, &cfg)
	}

	applyEnvOverrides(&cfg)
	return cfg
}

func loadYAML(path string, out interface{}) {
	f, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("failed to open config file %s: %v", path, err))
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(out); err != nil {
		panic(fmt.Sprintf("failed to decode config file %s: %v", path, err))
	}
}

func applyEnvOverrides(c *Config) {
	if v := os.Getenv("ADDR"); v != "" {
		c.Server.Addr = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Server.LogLevel = v
	}
	if v := os.Getenv("PG_USER"); v != "" {
		c.Postgres.User = v
	}
	if v := os.Getenv("PG_PASSWORD"); v != "" {
		c.Postgres.Password = v
	}
	if v := os.Getenv("PG_HOST"); v != "" {
		c.Postgres.Host = v
	}
	if v := os.Getenv("PG_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Postgres.Port)
	}
	if v := os.Getenv("PG_DBNAME"); v != "" {
		c.Postgres.DBName = v
	}
	if v := os.Getenv("PG_MAX_CONNS"); v != "" {
		fmt.Sscanf(v, "%d", &c.Postgres.MaxConns)
	}
	if v := os.Getenv("PG_SSLMODE"); v != "" {
		c.Postgres.SSLMode = v
	}
	if v := os.Getenv("LISTEN_CHANNEL"); v != "" {
		c.Listener.Channel = v
	}
	if v := os.Getenv("LISTEN_RECONNECT_SECONDS"); v != "" {
		fmt.Sscanf(v, "%d", &c.Listener.ReconnectSeconds)
	}
}

func (c Config) BuildDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Postgres.User,
		c.Postgres.Password,
		c.Postgres.Host,
		c.Postgres.Port,
		c.Postgres.DBName,
		c.Postgres.SSLMode,
	)
}