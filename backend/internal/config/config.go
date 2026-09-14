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
	RedisAddr      string
	WAQIToken      string
	WAQIBaseURL    string
}

func Load() Config {
	redisAddr := env("REDIS_ADDR", env("REDIS_HOST", "localhost")+":"+env("REDIS_PORT", "6379"))
	return Config{
		AppPort:        env("APP_PORT", "8000"),
		DBHost:         env("DB_HOST", "localhost"),
		DBPort:         env("DB_PORT", "5432"),
		DBUser:         env("DB_USER", "admin"),
		DBPassword:     env("DB_PASSWORD", "admin"),
		DBName:         env("DB_NAME", "aqi"),
		KeycloakIssuer: env("KEYCLOAK_ISSUER", "http://localhost:8080/realms/aqi-monitor"),
		RedisAddr:      redisAddr,
		WAQIToken:      env("WAQI_API_TOKEN", ""),
		WAQIBaseURL:    env("WAQI_BASE_URL", "https://api.waqi.info"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
