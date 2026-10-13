package database

import (
	"fmt"
	"log"

	"template-sqlmanager/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// BuildMySQLDSN genera una cadena de conexión para MySQL.
func BuildMySQLDSN(cfg *config.Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
}

// BuildPostgreSQLDSN genera una cadena de conexión para PostgreSQL.
func BuildPostgreSQLDSN(cfg *config.Config) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
}

// Connect conecta a la base de datos; tanto si es MySQL, como si es PostgreSQL.
func Connect(cfg *config.Config) *gorm.DB {
	var db *gorm.DB
	var err error

	// Se configura GORM según el entorno: Silencioso en prod, muestra las consultas SQL en dev
	gormConfig := &gorm.Config{Logger: getLogger(cfg.AppEnv)}

	// Se configura la conexión según el entorno: mysql (mariadb) o postgres
	switch cfg.DBDriver {
		case "mysql":
			dsn := BuildMySQLDSN(cfg)
			db, err = gorm.Open(mysql.Open(dsn), gormConfig)
		case "postgres":
			dsn := BuildPostgreSQLDSN(cfg)
			db, err = gorm.Open(postgres.Open(dsn), gormConfig)
		default:
			log.Fatalf("ERROR CRÍTICO: Driver '%s' no soportado.", cfg.DBDriver)
	}

	if err != nil {
		log.Fatalf("ERROR CRÍTICO: No se pudo conectar a %s (%s): %v", cfg.DBDriver, cfg.DBName, err)
	}

	log.Printf("Conexión a %s (%s) establecida exitosamente.", cfg.DBDriver, cfg.DBName)
	return db
}

// getLogger configura el nivel de información que muestran los logs según el entorno.
func getLogger(appEnv string) logger.Interface {
	if appEnv == "dev" {
		return logger.Default.LogMode(logger.Info) // Desarrollo (Muestra las consultas SQL)
	}
	return logger.Default.LogMode(logger.Error) // Producción (Silencioso)
}