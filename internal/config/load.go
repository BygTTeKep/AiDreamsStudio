package config

import (
	"time"

	"github.com/spf13/viper"
)

type DatabaseConfig struct {
	Dsn string `mapstructure:"dsn"`
}

type RedisConfig struct {
	Addr string `mapstructure:"addr"`
	Db   int    `mapstructure:"db"`
}

type AppConfig struct {
	Port int `mapstructure:"port"`
}

type JWTConfig struct {
	SecretKey    string `mapstructure:"secret_key"`
	ExpiresHours int    `mapstructure:"expires_hours"`
}

type StorageConfig struct {
	Type string `mapstructure:"type"`
	URL  string `mapstructure:"url"`
}

type ClaudeConfig struct {
	APIKey    string        `mapstructure:"api_key"` // пусто — берётся ANTHROPIC_API_KEY
	Model     string        `mapstructure:"model"`
	MaxTokens int64         `mapstructure:"max_tokens"`
	Effort    string        `mapstructure:"effort"`
	Workers   int           `mapstructure:"workers"`
	QueueSize int           `mapstructure:"queue_size"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

type Config struct {
	DB      DatabaseConfig `mapstructure:"database"`
	App     AppConfig      `mapstructure:"app"`
	Rdb     RedisConfig    `mapstructure:"redis"`
	JWT     JWTConfig      `mapstructure:"jwt"`
	Storage StorageConfig  `mapstructure:"storage"`
	Claude  ClaudeConfig   `mapstructure:"claude"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.AddConfigPath("../../internal/config/")
	v.SetConfigName("config")
	v.SetConfigType("yml")

	v.AutomaticEnv()

	var cfg Config

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
