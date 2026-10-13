package repositories

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// FieldChange representa el estado anterior y actual de un campo modificado.
type FieldChange struct {
	Anterior any `json:"anterior"`
	Actual   any `json:"actual"`
}

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

// Create inserta un nuevo registro y retorna su ID junto a los datos del registro (omitiendo id, *_at y *_by).
func (r *BaseRepository[T]) Create(entity *T) (string, map[string]any, error) {
	if err := r.db.Create(entity).Error; err != nil {
		return "", nil, err
	}

	id := extractEntityID(entity)
	data := extractBusinessData(entity)
	return id, data, nil
}

// Read busca un único registro utilizando su ID (UUID).
func (r *BaseRepository[T]) Read(id string) (*T, error) {
	var entity T
	err := r.db.First(&entity, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update actualiza los campos especificados y retorna el ID junto al diff de cambios (omitiendo id, *_at y *_by).
func (r *BaseRepository[T]) Update(id string, data *T) (string, map[string]FieldChange, error) {
	var anterior T
	if err := r.db.First(&anterior, "id = ?", id).Error; err != nil {
		return "", nil, fmt.Errorf("registro no encontrado: %w", err)
	}

	changes := extractFieldChanges(&anterior, data)
	if len(changes) == 0 {
		return id, changes, nil
	}

	if err := r.db.Model(&anterior).Updates(data).Error; err != nil {
		return "", nil, err
	}

	return id, changes, nil
}

// SoftDelete realiza un borrado lógico y retorna el ID junto a los datos del registro eliminado (omitiendo id, *_at y *_by).
func (r *BaseRepository[T]) SoftDelete(id string) (string, map[string]any, error) {
	var entity T
	if err := r.db.First(&entity, "id = ?", id).Error; err != nil {
		return "", nil, fmt.Errorf("registro no encontrado: %w", err)
	}

	data := extractBusinessData(&entity)

	if err := r.db.Delete(&entity).Error; err != nil {
		return "", nil, err
	}

	return id, data, nil
}

// HardDelete hace un borrado físico, eliminando permanentemente el registro de la base de datos.
func (r *BaseRepository[T]) HardDelete(id string) error {
	var entity T
	return r.db.Unscoped().Where("id = ?", id).Delete(&entity).Error
}

// --- Funciones Utilitarias ---

// extractEntityID obtiene el valor de ID de una entidad como string.
func extractEntityID[T any](entity *T) string {
	val := reflect.ValueOf(entity).Elem()
	idField := val.FieldByName("ID")
	if idField.IsValid() && !idField.IsZero() {
		return fmt.Sprintf("%v", idField.Interface())
	}
	return ""
}

// isSystemOrSensitiveField verifica si un campo debe excluirse de los datos de negocio y del diff de cambios.
func isSystemOrSensitiveField(field reflect.StructField) (bool, string) {
	if field.Anonymous {
		return true, ""
	}

	tagJSON := field.Tag.Get("json")
	parts := strings.Split(tagJSON, ",")
	fieldName := parts[0]

	// Campos sensibles marcados con json:"-"
	if fieldName == "-" {
		return true, ""
	}

	if fieldName == "" {
		fieldName = field.Name
	}

	// Campos del sistema: id, *_at, *_by
	if fieldName == "id" || strings.HasSuffix(fieldName, "_at") || strings.HasSuffix(fieldName, "_by") {
		return true, ""
	}

	return false, fieldName
}

// extractBusinessData extrae los campos puros de negocio en un map, omitiendo campos del sistema y sensibles.
func extractBusinessData[T any](entity *T) map[string]any {
	data := make(map[string]any)
	val := reflect.ValueOf(entity).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		ignore, fieldName := isSystemOrSensitiveField(field)
		if ignore {
			continue
		}
		data[fieldName] = val.Field(i).Interface()
	}

	return data
}

// extractFieldChanges compara dos entidades y retorna las diferencias omitiendo campos del sistema y sensibles.
func extractFieldChanges[T any](anterior, nuevo *T) map[string]FieldChange {
	changes := make(map[string]FieldChange)
	valAnterior := reflect.ValueOf(anterior).Elem()
	valNuevo := reflect.ValueOf(nuevo).Elem()
	typ := valAnterior.Type()

	for i := 0; i < valAnterior.NumField(); i++ {
		field := typ.Field(i)
		ignore, fieldName := isSystemOrSensitiveField(field)
		if ignore {
			continue
		}

		vAnt := valAnterior.Field(i).Interface()
		vNue := valNuevo.Field(i).Interface()

		if !valNuevo.Field(i).IsZero() && !reflect.DeepEqual(vAnt, vNue) {
			changes[fieldName] = FieldChange{
				Anterior: vAnt,
				Actual:   vNue,
			}
		}
	}

	return changes
}