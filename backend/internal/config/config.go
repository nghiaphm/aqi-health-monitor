package config

import "os"

// Config đọc cấu hình từ environment (do docker-compose/.env cung cấp).
type Config struct {
	AppPort           string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	KeycloakIssuer    string
	RedisAddr         string
	WAQIToken         string
	WAQIBaseURL       string
	CORSAllowedOrigin string
	ResendAPIKey      string
	ResendFromEmail   string
	ResendBaseURL     string
	VAPIDPublicKey    string
	VAPIDPrivateKey   string
	VAPIDSubject      string
}

func Load() Config {
	redisAddr := env("REDIS_ADDR", env("REDIS_HOST", "localhost")+":"+env("REDIS_PORT", "6379"))
	return Config{
		AppPort:           env("APP_PORT", "8000"),
		DBHost:            env("DB_HOST", "localhost"),
		DBPort:            env("DB_PORT", "5432"),
		DBUser:            env("DB_USER", "admin"),
		DBPassword:        env("DB_PASSWORD", "admin"),
		DBName:            env("DB_NAME", "aqi"),
		KeycloakIssuer:    env("KEYCLOAK_ISSUER", "http://localhost:8080/realms/aqi-monitor"),
		RedisAddr:         redisAddr,
		WAQIToken:         env("WAQI_API_TOKEN", ""),
		WAQIBaseURL:       env("WAQI_BASE_URL", "https://api.waqi.info"),
		CORSAllowedOrigin: env("CORS_ALLOWED_ORIGIN", "http://localhost:3000"),
		ResendAPIKey:      env("RESEND_API_KEY", ""),
		ResendFromEmail:   env("RESEND_FROM_EMAIL", "AQI Health Monitor <onboarding@resend.dev>"),
		ResendBaseURL:     env("RESEND_BASE_URL", "https://api.resend.com"),
		VAPIDPublicKey:    env("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:   env("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:      env("VAPID_SUBJECT", "mailto:alerts@aqi-health-monitor.local"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
