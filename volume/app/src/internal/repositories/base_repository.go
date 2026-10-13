package repositories

import (
	"gorm.io/gorm"
)

// BaseRepository es un repositorio genérico que provee métodos CRUD estándar para cualquier modelo.
// La [T any] significa que este repositorio puede tomar la forma de cualquier modelo (Ej: Usuario, Producto).
type BaseRepository[T any] struct {
	db *gorm.DB
}

// NewBaseRepository es el constructor. Recibe la conexión a la base de datos y devuelve el repositorio listo.
func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db: db,
	}
}

// Create inserta un nuevo registro en la base de datos.
func (r *BaseRepository[T]) Create(entity *T) error {
	return r.db.Create(entity).Error
}

// Read busca un único registro utilizando su ID (UUID).
func (r *BaseRepository[T]) Read(id string) (*T, error) {
	var entity T
	// First busca el primero que coincida. Al pasarle una estructura vacía 'entity', GORM la rellena.
	err := r.db.First(&entity, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update actualiza todos los campos de un registro existente.
func (r *BaseRepository[T]) Update(entity *T) error {
	// Save actualizará el registro si tiene ID, o lo insertará si no lo tiene.
	return r.db.Save(entity).Error
}

// SoftDelete hace un borrado lógico (Añade fecha al campo deleted_at pero no lo borra físicamente).
func (r *BaseRepository[T]) SoftDelete(id string) error {
	var entity T
	// Al llamar Delete sobre un modelo que tenga gorm.DeletedAt, GORM hace el SoftDelete automáticamente.
	return r.db.Where("id = ?", id).Delete(&entity).Error
}

// HardDelete hace un borrado físico, eliminando permanentemente el registro de la base de datos.
func (r *BaseRepository[T]) HardDelete(id string) error {
	var entity T
	// Unscoped() le dice a GORM que ignore el SoftDelete y destruya la fila de verdad.
	return r.db.Unscoped().Where("id = ?", id).Delete(&entity).Error
}