package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration (file + env overrides)
type Config struct {
	Server struct {
		Addr     string `mapstructure:"addr"`
		LogLevel string `mapstructure:"log_level"`
	} `mapstructure:"server"`

	Postgres struct {
		Host         string `mapstructure:"host"`
		Port         int    `mapstructure:"port"`
		User         string `mapstructure:"user"`
		Password     string `mapstructure:"password"`
		DBName       string `mapstructure:"db_name"`
		SSLMode      string `mapstructure:"ssl_mode"`
		MaxOpenConns int    `mapstructure:"max_open_conns"`
		MaxIdleConns int    `mapstructure:"max_idle_conns"`
	} `mapstructure:"postgres"`

	Listener struct {
		Channel          string `mapstructure:"channel"`
		ReconnectSeconds int    `mapstructure:"reconnect_seconds"`
	} `mapstructure:"listener"`
}

func Load() Config {
	v := viper.New()
	v.SetConfigName("application")
	v.SetConfigType("yaml")
	v.AddConfigPath("configs")
	_ = v.ReadInConfig() // optional; env can fully configure

	v.SetEnvPrefix("APP")
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("unable to decode config: %w", err))
	}
	validate(&cfg)
	return cfg
}

func validate(c *Config) {
	if c.Server.Addr == "" { c.Server.Addr = ":8080" }
	if c.Postgres.Port == 0 { c.Postgres.Port = 5432 }
	if c.Postgres.SSLMode == "" { c.Postgres.SSLMode = "disable" }
	if c.Postgres.MaxOpenConns == 0 { c.Postgres.MaxOpenConns = 10 }
	if c.Postgres.MaxIdleConns == 0 { c.Postgres.MaxIdleConns = 10 }
	if c.Listener.ReconnectSeconds <= 0 { c.Listener.ReconnectSeconds = 5 }
}

func (c Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Postgres.User,
		c.Postgres.Password,
		c.Postgres.Host,
		c.Postgres.Port,
		c.Postgres.DBName,
		c.Postgres.SSLMode,
	)
}

func (c Config) Backoff() time.Duration { return time.Duration(c.Listener.ReconnectSeconds) * time.Second }