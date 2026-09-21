// Package config 负责从环境变量加载运行配置。
//
// 所有配置项都有开发环境可用的默认值，因此不加任何环境变量也能直接启动。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 支持的数据库驱动。
const (
	DriverPostgres = "postgres"
	DriverSQLite   = "sqlite"
)

// Config 服务运行配置。
type Config struct {
	AppEnv   string
	HTTPPort string
	LogLevel string

	DBDriver   string
	DBPath     string
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string
	DBTimeZone string

	SeedEnabled bool
}

// Load 读取环境变量并返回配置。
func Load() *Config {
	return &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		HTTPPort:    getEnv("APP_PORT", "8080"),
		LogLevel:    getEnv("APP_LOG_LEVEL", "info"),
		DBDriver:    strings.ToLower(getEnv("DB_DRIVER", DriverPostgres)),
		DBPath:      getEnv("DB_PATH", filepath.Join("data", "desilting.db")),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBName:      getEnv("DB_NAME", "desilting"),
		DBUser:      getEnv("DB_USER", "desilting"),
		DBPassword:  getEnv("DB_PASSWORD", "desilting123456"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
		DBTimeZone:  getEnv("DB_TIMEZONE", "Asia/Shanghai"),
		SeedEnabled: getBoolEnv("APP_SEED_ENABLED", true),
	}
}

// PostgresDSN 构造 PostgreSQL 连接串。
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode, c.DBTimeZone,
	)
}

// Describe 返回用于启动日志的简要描述。
func (c *Config) Describe() string {
	if c.DBDriver == DriverSQLite {
		return fmt.Sprintf("%s(%s)", c.DBDriver, c.DBPath)
	}
	return fmt.Sprintf("%s(%s:%s/%s)", c.DBDriver, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
