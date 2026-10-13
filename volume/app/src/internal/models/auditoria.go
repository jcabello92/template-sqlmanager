package models

import (
	"time"

	"github.com/google/uuid"
)

// Auditoria contiene la estructura de la tabla homónima.
type Auditoria struct {
	PrimaryKey

	TipoTransaccion string     `gorm:"type:varchar(10);not null" json:"tipo_transaccion"` // "INSERT" | "UPDATE" | "DELETE"
	TablaSlug       string     `gorm:"type:varchar(50);not null" json:"tabla_slug"`
	RegistroID      uuid.UUID  `gorm:"type:char(36);not null" json:"registro_id"`
	Modificaciones  string     `gorm:"type:json;not null" json:"modificaciones"` // Se usa 'json' para compatibilidad MySQL/PostgreSQL. Si es estrictamente PostgreSQL se puede cambiar por 'jsonb'
	AutorID         *uuid.UUID `gorm:"type:char(36)" json:"autor_id"` // NULL si fue el sistema
	AutorIP         *string    `gorm:"type:varchar(45)" json:"autor_ip"` // Se usa varchar(45) para IPv6 y para compatibilidad MySQL/PostgreSQL. Si es estrictamente PostgreSQL se puede cambiar por 'inet'
	UserAgent       *string    `gorm:"type:text" json:"user_agent"`
	CreatedAt       time.Time  `gorm:"not null" json:"created_at"`
}

// TableName sobreescribe el nombre de la tabla para adaptarse al nombre real que esta tiene en la base de datos.
func (Auditoria) TableName() string {
	return "auditoria"
}