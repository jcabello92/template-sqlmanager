package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PrimaryKey contiene el ID de las tablas.
type PrimaryKey struct {
	ID uuid.UUID `gorm:"type:char(36);primaryKey;not null" json:"id"` // Se usa char(36) para compatibilidad con MySQL/MariaDB y PostgreSQL
}

// Base contiene la estructura base para las tablas.
type Base struct {
	PrimaryKey

	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CreatedBy *uuid.UUID     `gorm:"type:char(36)" json:"-"`
	UpdatedBy *uuid.UUID     `gorm:"type:char(36)" json:"-"`
	DeletedBy *uuid.UUID     `gorm:"type:char(36)" json:"-"`
}

// BeforeCreate es un Hook de GORM que se dispara automáticamente antes de hacer un INSERT en la base de datos; asegurandose de que siempre que se cree un registro, se haga con un ID (UUIDv7).
func (p *PrimaryKey) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID, err = uuid.NewV7()
		if err != nil {
			return err
		}
	}
	return
}