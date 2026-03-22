package config

import (
	"fmt"
	"time"
)

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret           string        `mapstructure:"jwt_secret"`
	ExpiresIn        time.Duration `mapstructure:"jwt_expires_in"`
	RefreshExpiresIn time.Duration `mapstructure:"jwt_refresh_expires_in"`
	Issuer           string        `mapstructure:"jwt_issuer"`
}

// GetJWTSecret returns the configured signing secret.
func GetJWTSecret() (string, error) {
	if AppConfig == nil {
		return "", fmt.Errorf("config not loaded")
	}
	if AppConfig.JWT.Secret == "" {
		return "", fmt.Errorf("JWT_SECRET not configured")
	}
	return AppConfig.JWT.Secret, nil
}

// GetJWTExpiresInTime returns access-token lifetime.
func GetJWTExpiresInTime() time.Duration {
	if AppConfig == nil || AppConfig.JWT.ExpiresIn == 0 {
		return 24 * time.Hour
	}
	return AppConfig.JWT.ExpiresIn
}

// GetJWTRefreshExpiresIn returns refresh-token lifetime.
func GetJWTRefreshExpiresIn() time.Duration {
	if AppConfig == nil || AppConfig.JWT.RefreshExpiresIn == 0 {
		return 7 * 24 * time.Hour
	}
	return AppConfig.JWT.RefreshExpiresIn
}

// GetJWTIssuer returns the JWT issuer claim default.
func GetJWTIssuer() string {
	return AppConfig.JWT.Issuer
}
