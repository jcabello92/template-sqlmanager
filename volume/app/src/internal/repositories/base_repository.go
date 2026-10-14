package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"template-sqlmanager/internal/database"
	"template-sqlmanager/internal/interceptors"
	"template-sqlmanager/internal/models"
)

// Tabler define el contrato que debe implementar cualquier modelo para indicar explícitamente el nombre de su tabla.
type Tabler interface {
	TableName() string
}

// FieldChange representa el estado anterior y actual de un campo modificado para la auditoría.
type FieldChange struct {
	Anterior any `json:"anterior"`
	Actual   any `json:"actual"`
}

// BaseRepository es un repositorio genérico que provee métodos CRUD estándar con auditoría automática y transaccional.
type BaseRepository[T any] struct {
	db *gorm.DB
}

// NewBaseRepository es el constructor. Recibe la conexión a la base de datos y devuelve el repositorio listo.
func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db: db,
	}
}

// Create inserta un nuevo registro, registra automáticamente la auditoría (INSERT) y retorna el ID del nuevo registro.
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) (string, error) {
	if entity == nil {
		return "", errors.New("la entidad proporcionada no puede ser nula")
	}

	tablaSlug, err := r.getTableName()
	if err != nil {
		return "", err
	}

	auditData := interceptors.GetAuditFromContext(ctx)
	setAuditField(entity, "CreatedBy", auditData.AutorID)

	var idStr string
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(entity).Error; err != nil {
			return fmt.Errorf("error al insertar el registro en la base de datos: %w", err)
		}

		idStr = extractEntityID(entity)
		if idStr == "" {
			return errors.New("no se pudo obtener el ID del registro creado")
		}

		regUUID, err := uuid.Parse(idStr)
		if err != nil {
			return fmt.Errorf("el ID del registro creado no es un UUID válido: %w", err)
		}

		modificaciones := extractCreateChanges(entity)
		if err := recordAudit(tx, ctx, "INSERT", tablaSlug, regUUID, modificaciones); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return idStr, nil
}

// Read busca un único registro utilizando su ID (UUID).
func (r *BaseRepository[T]) Read(ctx context.Context, id string) (*T, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("el identificador (id) no puede estar vacío")
	}

	var entity T
	if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("error al buscar el registro con ID %s: %w", id, err)
	}
	return &entity, nil
}

// FindOne busca un único registro utilizando filtros dinámicos y retorna el primer resultado coincidente.
func (r *BaseRepository[T]) FindOne(ctx context.Context, filters []database.Filter) (*T, error) {
	if len(filters) == 0 {
		return nil, errors.New("se debe proporcionar al menos un filtro para la búsqueda")
	}

	var entity T
	query := r.db.WithContext(ctx).Model(&entity)

	var err error
	query, err = database.ApplyFilters(query, filters)
	if err != nil {
		return nil, fmt.Errorf("error al aplicar filtros en la búsqueda: %w", err)
	}

	if err := query.First(&entity).Error; err != nil {
		return nil, fmt.Errorf("registro no encontrado para los criterios de búsqueda especificados: %w", err)
	}

	return &entity, nil
}

// Exists comprueba de manera eficiente si un registro existe y se encuentra activo en la base de datos por su ID.
func (r *BaseRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, errors.New("el identificador (id) no puede estar vacío")
	}

	if _, err := uuid.Parse(id); err != nil {
		return false, fmt.Errorf("el identificador (id) '%s' no es un UUID válido: %w", id, err)
	}

	var count int64
	var entity T
	if err := r.db.WithContext(ctx).Model(&entity).Where("id = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, fmt.Errorf("error al verificar la existencia del registro con ID %s: %w", id, err)
	}

	return count > 0, nil
}

// List consulta una colección de registros con paginación, filtros dinámicos (AND/OR, anidados) y ordenamiento.
func (r *BaseRepository[T]) List(ctx context.Context, params database.ListParams) ([]T, *database.PaginationMetadata, error) {
	params.Normalize()

	if !database.IsValidFieldName(params.OrderBy) {
		return nil, nil, fmt.Errorf("campo de ordenamiento no válido: '%s'", params.OrderBy)
	}

	var entity T
	query := r.db.WithContext(ctx).Model(&entity)

	var err error
	query, err = database.ApplyFilters(query, params.Filters)
	if err != nil {
		return nil, nil, fmt.Errorf("error al aplicar filtros en el listado: %w", err)
	}

	var totalRecords int64
	if err := query.Count(&totalRecords).Error; err != nil {
		return nil, nil, fmt.Errorf("error al contar los registros para el listado: %w", err)
	}

	totalPages := 0
	if totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(params.PageSize)))
	}

	metadata := &database.PaginationMetadata{
		CurrentPage:  params.Page,
		PageSize:     params.PageSize,
		TotalPages:   totalPages,
		TotalRecords: totalRecords,
	}

	items := make([]T, 0)
	if totalRecords == 0 {
		return items, metadata, nil
	}

	orderClause := fmt.Sprintf("%s %s", params.OrderBy, params.OrderDir)
	offset := (params.Page - 1) * params.PageSize

	if err := query.Order(orderClause).Limit(params.PageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, nil, fmt.Errorf("error al obtener los registros del listado: %w", err)
	}

	return items, metadata, nil
}

// Update actualiza los campos especificados, registra la auditoría (UPDATE) y retorna el ID del registro actualizado.
func (r *BaseRepository[T]) Update(ctx context.Context, id string, data *T) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("el identificador (id) no puede estar vacío")
	}
	if data == nil {
		return "", errors.New("los datos para actualizar no pueden ser nulos")
	}

	tablaSlug, err := r.getTableName()
	if err != nil {
		return "", err
	}

	regUUID, err := uuid.Parse(id)
	if err != nil {
		return "", fmt.Errorf("el identificador (id) '%s' no es un UUID válido: %w", id, err)
	}

	auditData := interceptors.GetAuditFromContext(ctx)

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var anterior T
		if err := tx.First(&anterior, "id = ?", id).Error; err != nil {
			return fmt.Errorf("registro no encontrado para actualizar con ID %s: %w", id, err)
		}

		changes := extractFieldChanges(&anterior, data)
		if len(changes) == 0 {
			return nil
		}

		setAuditField(data, "UpdatedBy", auditData.AutorID)

		if err := tx.Model(&anterior).Updates(data).Error; err != nil {
			return fmt.Errorf("error al actualizar el registro con ID %s: %w", id, err)
		}

		if err := recordAudit(tx, ctx, "UPDATE", tablaSlug, regUUID, changes); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return id, nil
}

// SoftDelete realiza un borrado lógico, registra la auditoría (DELETE) y retorna el ID del registro eliminado.
func (r *BaseRepository[T]) SoftDelete(ctx context.Context, id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("el identificador (id) no puede estar vacío")
	}

	tablaSlug, err := r.getTableName()
	if err != nil {
		return "", err
	}

	regUUID, err := uuid.Parse(id)
	if err != nil {
		return "", fmt.Errorf("el identificador (id) '%s' no es un UUID válido: %w", id, err)
	}

	auditData := interceptors.GetAuditFromContext(ctx)

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var entity T
		if err := tx.First(&entity, "id = ?", id).Error; err != nil {
			return fmt.Errorf("registro no encontrado para borrado lógico con ID %s: %w", id, err)
		}

		modificaciones := extractDeleteChanges(&entity)

		if auditData.AutorID != nil {
			_ = tx.Model(&entity).Update("deleted_by", auditData.AutorID).Error
		}

		if err := tx.Delete(&entity).Error; err != nil {
			return fmt.Errorf("error al realizar el borrado lógico del registro con ID %s: %w", id, err)
		}

		if err := recordAudit(tx, ctx, "DELETE", tablaSlug, regUUID, modificaciones); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return id, nil
}

// Restore reactiva un registro previamente borrado lógicamente, registra la auditoría (RESTORE) y retorna su ID.
func (r *BaseRepository[T]) Restore(ctx context.Context, id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("el identificador (id) no puede estar vacío")
	}

	tablaSlug, err := r.getTableName()
	if err != nil {
		return "", err
	}

	regUUID, err := uuid.Parse(id)
	if err != nil {
		return "", fmt.Errorf("el identificador (id) '%s' no es un UUID válido: %w", id, err)
	}

	auditData := interceptors.GetAuditFromContext(ctx)

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var entity T
		if err := tx.Unscoped().First(&entity, "id = ?", id).Error; err != nil {
			return fmt.Errorf("registro no encontrado para restaurar con ID %s: %w", id, err)
		}

		if !isSoftDeleted(&entity) {
			return fmt.Errorf("el registro con ID %s no se encuentra borrado lógicamente", id)
		}

		// Como contraparte exacta del SoftDelete, en RESTORE se registra anterior: null y actual: datos_guardados
		modificaciones := extractCreateChanges(&entity)

		updates := map[string]any{
			"deleted_at": nil,
			"deleted_by": nil,
		}
		if auditData.AutorID != nil {
			updates["updated_by"] = auditData.AutorID
		}

		if err := tx.Unscoped().Model(&entity).Updates(updates).Error; err != nil {
			return fmt.Errorf("error al restaurar el registro con ID %s: %w", id, err)
		}

		if err := recordAudit(tx, ctx, "RESTORE", tablaSlug, regUUID, modificaciones); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return id, nil
}

// HardDelete hace un borrado físico permanente del registro y de su historial de auditoría asociado (útil para Sagas y Rollbacks).
func (r *BaseRepository[T]) HardDelete(ctx context.Context, id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("el identificador (id) no puede estar vacío")
	}

	regUUID, err := uuid.Parse(id)
	if err != nil {
		return "", fmt.Errorf("el identificador (id) '%s' no es un UUID válido: %w", id, err)
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var entity T
		result := tx.Unscoped().Where("id = ?", id).Delete(&entity)
		if result.Error != nil {
			return fmt.Errorf("error al eliminar permanentemente el registro con ID %s: %w", id, result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("no se encontró ningún registro para eliminar permanentemente con ID %s", id)
		}

		// Borrado físico de los registros de auditoría asociados para evitar registros huérfanos tras un rollback/saga
		if err := tx.Unscoped().Where("registro_id = ?", regUUID).Delete(&models.Auditoria{}).Error; err != nil {
			return fmt.Errorf("error al eliminar la auditoría del registro con ID %s: %w", id, err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Registro con ID %s eliminado permanentemente con éxito", id), nil
}

// --- Funciones Utilitarias Internas ---

// isSoftDeleted verifica si la entidad tiene el campo DeletedAt activo indicando borrado lógico.
func isSoftDeleted[T any](entity *T) bool {
	if entity == nil {
		return false
	}
	val := reflect.ValueOf(entity).Elem()
	if !val.IsValid() {
		return false
	}
	delField := val.FieldByName("DeletedAt")
	if delField.IsValid() {
		if delAt, ok := delField.Interface().(gorm.DeletedAt); ok {
			return delAt.Valid
		}
	}
	return true
}

// getTableName valida que el modelo implemente el método TableName() string y retorna su valor.
func (r *BaseRepository[T]) getTableName() (string, error) {
	var entity T
	if tabler, ok := any(entity).(Tabler); ok {
		name := strings.TrimSpace(tabler.TableName())
		if name != "" {
			return name, nil
		}
	}
	if tabler, ok := any(&entity).(Tabler); ok {
		name := strings.TrimSpace(tabler.TableName())
		if name != "" {
			return name, nil
		}
	}
	return "", fmt.Errorf("el modelo %T no implementa el método requerido TableName() string", entity)
}

// recordAudit inserta un registro en la tabla auditoria dentro de la transacción activa.
func recordAudit(
	tx *gorm.DB,
	ctx context.Context,
	tipoTransaccion string,
	tablaSlug string,
	registroID uuid.UUID,
	modificaciones map[string]FieldChange,
) error {
	if len(modificaciones) == 0 && tipoTransaccion == "UPDATE" {
		return nil
	}

	jsonBytes, err := json.Marshal(modificaciones)
	if err != nil {
		return fmt.Errorf("error al serializar modificaciones de auditoría: %w", err)
	}

	auditData := interceptors.GetAuditFromContext(ctx)

	audit := models.Auditoria{
		TipoTransaccion: tipoTransaccion,
		TablaSlug:       tablaSlug,
		RegistroID:      registroID,
		Modificaciones:  string(jsonBytes),
		AutorID:         auditData.AutorID,
		AutorIP:         auditData.AutorIP,
		UserAgent:       auditData.UserAgent,
	}

	if err := tx.WithContext(ctx).Create(&audit).Error; err != nil {
		return fmt.Errorf("error al registrar auditoría (%s en %s): %w", tipoTransaccion, tablaSlug, err)
	}

	return nil
}

// setAuditField asigna el AutorID a un campo de autoría (CreatedBy, UpdatedBy, DeletedBy) mediante reflexión si existe en la entidad.
func setAuditField[T any](entity *T, fieldName string, autorID *uuid.UUID) {
	if entity == nil || autorID == nil {
		return
	}
	val := reflect.ValueOf(entity).Elem()
	if !val.IsValid() {
		return
	}
	field := val.FieldByName(fieldName)
	if field.IsValid() && field.CanSet() && field.Type() == reflect.TypeOf((*uuid.UUID)(nil)) {
		field.Set(reflect.ValueOf(autorID))
	}
}

// extractEntityID obtiene el valor de ID de una entidad como string.
func extractEntityID[T any](entity *T) string {
	if entity == nil {
		return ""
	}
	val := reflect.ValueOf(entity).Elem()
	if !val.IsValid() {
		return ""
	}
	idField := val.FieldByName("ID")
	if idField.IsValid() && !idField.IsZero() {
		return fmt.Sprintf("%v", idField.Interface())
	}
	return ""
}

// isSystemOrSensitiveField verifica si un campo debe excluirse de los datos de negocio y del diff de cambios.
func isSystemOrSensitiveField(field reflect.StructField) (bool, string) {
	if field.Anonymous || !field.IsExported() {
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

// extractCreateChanges construye el mapa de modificaciones para INSERT: {"campo": {"anterior": null, "actual": valor}}.
func extractCreateChanges[T any](entity *T) map[string]FieldChange {
	changes := make(map[string]FieldChange)
	if entity == nil {
		return changes
	}
	val := reflect.ValueOf(entity).Elem()
	if !val.IsValid() {
		return changes
	}
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		ignore, fieldName := isSystemOrSensitiveField(field)
		if ignore {
			continue
		}
		fieldVal := val.Field(i)
		if fieldVal.CanInterface() {
			changes[fieldName] = FieldChange{
				Anterior: nil,
				Actual:   fieldVal.Interface(),
			}
		}
	}

	return changes
}

// extractDeleteChanges construye el mapa de modificaciones para DELETE: {"campo": {"anterior": valor, "actual": null}}.
func extractDeleteChanges[T any](entity *T) map[string]FieldChange {
	changes := make(map[string]FieldChange)
	if entity == nil {
		return changes
	}
	val := reflect.ValueOf(entity).Elem()
	if !val.IsValid() {
		return changes
	}
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		ignore, fieldName := isSystemOrSensitiveField(field)
		if ignore {
			continue
		}
		fieldVal := val.Field(i)
		if fieldVal.CanInterface() {
			changes[fieldName] = FieldChange{
				Anterior: fieldVal.Interface(),
				Actual:   nil,
			}
		}
	}

	return changes
}

// extractFieldChanges compara dos entidades y retorna las diferencias para UPDATE: {"campo": {"anterior": anterior, "actual": actual}}.
func extractFieldChanges[T any](anterior, nuevo *T) map[string]FieldChange {
	changes := make(map[string]FieldChange)
	if anterior == nil || nuevo == nil {
		return changes
	}
	valAnterior := reflect.ValueOf(anterior).Elem()
	valNuevo := reflect.ValueOf(nuevo).Elem()
	if !valAnterior.IsValid() || !valNuevo.IsValid() {
		return changes
	}
	typ := valAnterior.Type()

	for i := 0; i < valAnterior.NumField(); i++ {
		field := typ.Field(i)
		ignore, fieldName := isSystemOrSensitiveField(field)
		if ignore {
			continue
		}

		fAnt := valAnterior.Field(i)
		fNue := valNuevo.Field(i)

		if !fAnt.CanInterface() || !fNue.CanInterface() {
			continue
		}

		vAnt := fAnt.Interface()
		vNue := fNue.Interface()

		if !fNue.IsZero() && !reflect.DeepEqual(vAnt, vNue) {
			changes[fieldName] = FieldChange{
				Anterior: vAnt,
				Actual:   vNue,
			}
		}
	}

	return changes
}