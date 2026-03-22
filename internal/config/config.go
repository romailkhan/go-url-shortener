package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Port        string         `mapstructure:"port"`
	Database    DatabaseConfig `mapstructure:",squash"`
	JWT         JWTConfig      `mapstructure:",squash"`
	Redis       RedisConfig    `mapstructure:",squash"`
	Environment string         `mapstructure:"environment"`
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Host        string `mapstructure:"db_host"`
	Port        string `mapstructure:"db_port"`
	User        string `mapstructure:"db_user"`
	Password    string `mapstructure:"db_password"`
	Name        string `mapstructure:"db_name"`
	SSLMode     string `mapstructure:"db_sslmode"`
	Environment string `mapstructure:"db_environment"`
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Host     string `mapstructure:"redis_host"`
	Port     string `mapstructure:"redis_port"`
	Password string `mapstructure:"redis_password"`
}

// AppConfig is the global config instance set by LoadConfig.
var AppConfig *Config

// LoadConfig initializes and loads configuration from environment variables and .env file.
func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No .env file found: %v", err)
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	AppConfig = &config
	return &config, nil
}

// GetPort returns the configured port number (no leading colon), defaulting to 8081.
func GetPort() string {
	if AppConfig == nil {
		return "8081"
	}
	p := strings.TrimSpace(AppConfig.Port)
	if p == "" {
		return "8081"
	}
	return p
}

// GetDatabaseURL returns the database connection URL.
func GetDatabaseURL() (string, error) {
	if AppConfig == nil {
		return "", fmt.Errorf("config not loaded")
	}

	if AppConfig.Database.User == "" || AppConfig.Database.Password == "" || AppConfig.Database.Name == "" {
		return "", fmt.Errorf("missing required database configuration: user, password, or name")
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		AppConfig.Database.User,
		AppConfig.Database.Password,
		AppConfig.Database.Host,
		AppConfig.Database.Port,
		AppConfig.Database.Name,
		AppConfig.Database.SSLMode,
	), nil
}
