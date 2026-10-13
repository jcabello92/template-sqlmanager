package config

import (
    "log"
    "os"

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

// GetEnvOrCrash busca una variable de entorno; si no existe, finaliza el programa con un error.
func GetEnvOrCrash(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("ERROR CRÍTICO: La variable de entorno '%s' es obligatoria y no está definida.", key)
	}
	return value
}