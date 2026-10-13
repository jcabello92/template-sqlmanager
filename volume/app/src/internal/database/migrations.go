package database

import (
	"log"

	"template-sqlmanager/internal/models"

	"gorm.io/gorm"
)

// RunMigrations ejecuta las migraciones para crear o actualizar las tablas en la base de datos.
func RunMigrations(db *gorm.DB) {
	log.Println("Ejecutando migraciones en la base de datos...")

	err := db.AutoMigrate(
		// --- MODELOS DEL SISTEMA (No borrar) ---
		&models.Auditoria{}, // Esta tabla debe existir obligatoriamente en la base de datos

		// --- MODELOS DEL NEGOCIO ---
		// NOTA: Cuando clones esta plantilla, agrega aquí los modelos del proyecto
		// Ejemplo:
		// &models.Usuario{},
		// &models.Producto{},
	)

	if err != nil {
		log.Fatalf("ERROR CRÍTICO: Fallo al ejecutar migraciones: %v", err)
	}

	log.Println("Ejecución completada exitosamente.")
}