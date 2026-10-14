package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv     string
	GRPCHost   string
	GRPCPort   string
	
	DBDriver   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Pool de conexiones
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime int // en minutos
}

// LoadEnv carga las variables de entorno al sistema.
func LoadEnv() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("ADVERTENCIA: No se encontró el archivo con las variables de entorno. Se intentarán leer desde el entorno del sistema")
	}

	cfg := &Config{
		AppEnv:     GetEnvOrDefault("APP_ENV", "prod"),
		GRPCHost:   GetEnvOrDefault("GRPC_HOST", "0.0.0.0"),
		GRPCPort:   GetEnvOrDefault("GRPC_PORT", "50051"),

		DBDriver:   GetEnvOrDefault("DB_DRIVER", "postgres"),
		DBHost:     GetEnvOrCrash("DB_HOST"),
		DBPort:     GetEnvOrDefault("DB_PORT", "5432"),
		DBUser:     GetEnvOrCrash("DB_USER"),
		DBPassword: GetEnvOrCrash("DB_PASSWORD"),
		DBName:     GetEnvOrCrash("DB_NAME"),

		DBMaxOpenConns:    GetEnvIntOrDefault("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    GetEnvIntOrDefault("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: GetEnvIntOrDefault("DB_CONN_MAX_LIFETIME", 15),
	}

	return cfg
}

// GetEnvOrDefault busca una variable de entorno; si no existe, devuelve un valor por defecto.
func GetEnvOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

// GetEnvIntOrDefault busca una variable de entorno numérica entera; si no existe o no es válida, devuelve un valor por defecto.
func GetEnvIntOrDefault(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if intVal, err := strconv.Atoi(value); err == nil && intVal > 0 {
			return intVal
		}
	}
	return fallback
}

// GetEnvOrCrash busca una variable de entorno; si no existe, finaliza el programa con un error.
func GetEnvOrCrash(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("ERROR CRÍTICO: La variable de entorno '%s' es obligatoria y no está definida.", key)
	}
	return value
}