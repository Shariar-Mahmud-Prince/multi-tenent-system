package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all service configuration options.
type Config struct {
	ServerPort         string
	Env                string
	BaseDomain         string
	DatabaseURL        string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration
	RedisURL           string
	JWTSecret          string
	ServerIP           string
	MigrationsDir      string
	AllowedSystemHosts []string
}

// LoadConfig reads configuration from the environment with robust fallbacks.
func LoadConfig() *Config {
	baseDomain := getEnv("BASE_DOMAIN", "yourplatform.local")
	baseDomain = strings.ToLower(strings.TrimSpace(baseDomain))

	return &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		BaseDomain:         baseDomain,
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/tenant_domains?sslmode=disable"),
		DBMaxOpenConns:     getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:     getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime:  getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		RedisURL:           getEnv("REDIS_URL", "redis://redis:6379/0"),
		JWTSecret:          getEnv("JWT_SECRET", "super-secret-production-grade-jwt-key-32bytes!"),
		ServerIP:           getEnv("SERVER_IP", "127.0.0.1"),
		MigrationsDir:      getEnv("MIGRATIONS_DIR", "migrations"),
		AllowedSystemHosts: []string{"localhost", "127.0.0.1", baseDomain, "api." + baseDomain, "monitor." + baseDomain},
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(key, "")
	if val, err := time.ParseDuration(valStr); err == nil {
		return val
	}
	return defaultVal
}
