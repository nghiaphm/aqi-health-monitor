package config

import "os"

// Config đọc cấu hình từ environment (do docker-compose/.env cung cấp).
type Config struct {
	AppPort        string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	KeycloakIssuer string
}

func Load() Config {
	return Config{
		AppPort:        env("APP_PORT", "8000"),
		DBHost:         env("DB_HOST", "localhost"),
		DBPort:         env("DB_PORT", "5432"),
		DBUser:         env("DB_USER", "admin"),
		DBPassword:     env("DB_PASSWORD", "admin"),
		DBName:         env("DB_NAME", "aqi"),
		KeycloakIssuer: env("KEYCLOAK_ISSUER", "http://localhost:8080/realms/aqi-monitor"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
