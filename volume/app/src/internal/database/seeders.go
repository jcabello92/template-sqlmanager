package database

import (
	"log"

	"gorm.io/gorm"
)

// RunSeeders ejecuta las inserciones de los datos iniciales en la base de datos.
func RunSeeders(db *gorm.DB) {
	log.Println("Ejecutando inserción de datos iniciales en la base de datos...")

	// --- DATOS DEL NEGOCIO ---
	// NOTA: Cuando clones esta plantilla, agrega aquí los datos iniciales del proyecto
	// Ejemplo:
	// usuario := models.Usuario{
	// 	Nombre: "Juan Pérez", 
	// 	CorreoElectronico: "juan.perez@correo.com",
	// }
	//
	// Busca por CorreoElectronico. Si no lo encuentra, lo inserta.
	// result := db.Where(models.Usuario{CorreoElectronico: "juan.perez@correo.com"}).FirstOrCreate(&usuario)
	//
	// if result.RowsAffected > 0 {
	// 	log.Printf("Registro insertado exitosamente: %s (%s).\n", usuario.Nombre, usuario.CorreoElectronico)
	// }
	// else {
	//  log.Printf("Dato omitido: El usuario %s (%s) ya existía en la base de datos.\n", usuario.Nombre, usuario.CorreoElectronico)
	// }


	log.Println("Ejecución completada exitosamente.")
}