package database

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var validFieldRegex = regexp.MustCompile(`^[a-zA-Z0-9_.]+$`)

// Filter representa una regla de filtrado individual o un grupo anidado de filtros.
// El campo Logic actúa como conector infijo hacia adelante: define cómo se conecta este elemento
// con el elemento que viene inmediatamente después a su mismo nivel ("AND" u "OR").
// Para el último elemento de una lista o grupo, Logic se deja vacío o null ya que no hay nadie después.
type Filter struct {
	Field    string   `json:"field"`    // Nombre del campo (ej: "nombre", "estado", "precio")
	Operator string   `json:"operator"` // "=", "!=", ">", ">=", "<", "<=", "LIKE", "ILIKE", "IN", "NOT IN", "BETWEEN", "IS NULL", "IS NOT NULL"
	Value    any      `json:"value"`    // Valor a comparar
	Logic    string   `json:"logic"`    // Conector con el siguiente elemento a su mismo nivel: "AND" u "OR" (el último de la lista se deja vacío)
	Group    []Filter `json:"group"`    // Subfiltros anidados para generar paréntesis en SQL: (A OR B)
}

// PaginationMetadata contiene la información estandarizada de paginación.
type PaginationMetadata struct {
	CurrentPage  int   `json:"current_page"`
	PageSize     int   `json:"page_size"`
	TotalPages   int   `json:"total_pages"`
	TotalRecords int64 `json:"total_records"`
}

// ListParams encapsula los parámetros de paginación, ordenamiento y filtrado para consultas de colecciones.
type ListParams struct {
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	OrderBy  string   `json:"order_by"`
	OrderDir string   `json:"order_dir"`
	Filters  []Filter `json:"filters"`
}

// Normalize asegura que los parámetros de consulta tengan valores válidos y seguros.
func (p *ListParams) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	} else if p.PageSize > 100 {
		p.PageSize = 100
	}

	p.OrderDir = strings.ToUpper(strings.TrimSpace(p.OrderDir))
	if p.OrderDir != "ASC" && p.OrderDir != "DESC" {
		p.OrderDir = "ASC"
	}

	p.OrderBy = strings.TrimSpace(p.OrderBy)
	if p.OrderBy == "" {
		p.OrderBy = "id"
	}
}

// IsValidFieldName comprueba si un nombre de campo contiene únicamente caracteres seguros (alfanuméricos, puntos y guiones bajos).
func IsValidFieldName(field string) bool {
	return validFieldRegex.MatchString(field)
}

// ApplyFilters aplica recursivamente los filtros a la consulta GORM.
// Cada elemento utiliza el Logic del elemento anterior como su conector de entrada ("AND" u "OR"),
// permitiendo una sintaxis infija limpia y sin duplicidades ni a nivel plano ni en subgrupos anidados.
func ApplyFilters(db *gorm.DB, filters []Filter) (*gorm.DB, error) {
	if len(filters) == 0 {
		return db, nil
	}

	nextConnector := "AND"
	isFirst := true

	for _, f := range filters {
		// 1. Si es un grupo anidado de filtros (genera paréntesis en SQL)
		if len(f.Group) > 0 {
			subDB := db.Session(&gorm.Session{NewDB: true})
			appliedSubDB, err := ApplyFilters(subDB, f.Group)
			if err != nil {
				return nil, err
			}

			if isFirst {
				db = db.Where(appliedSubDB)
				isFirst = false
			} else if nextConnector == "OR" {
				db = db.Or(appliedSubDB)
			} else {
				db = db.Where(appliedSubDB)
			}

			// El Logic de este grupo define cómo se une el siguiente elemento al mismo nivel
			logic := strings.ToUpper(strings.TrimSpace(f.Logic))
			if logic == "OR" {
				nextConnector = "OR"
			} else {
				nextConnector = "AND"
			}
			continue
		}

		// 2. Si es una regla simple de filtrado
		field := strings.TrimSpace(f.Field)
		if field == "" {
			continue
		}

		if !IsValidFieldName(field) {
			return nil, fmt.Errorf("nombre de campo no válido para filtrado: '%s'", field)
		}

		clause, args, err := buildCondition(field, f.Operator, f.Value)
		if err != nil {
			return nil, err
		}

		if isFirst {
			db = db.Where(clause, args...)
			isFirst = false
		} else if nextConnector == "OR" {
			db = db.Or(clause, args...)
		} else {
			db = db.Where(clause, args...)
		}

		// El Logic de este filtro define cómo se une el siguiente elemento a su mismo nivel
		logic := strings.ToUpper(strings.TrimSpace(f.Logic))
		if logic == "OR" {
			nextConnector = "OR"
		} else {
			nextConnector = "AND"
		}
	}

	return db, nil
}

// buildCondition construye la cláusula SQL y los argumentos según el operador.
func buildCondition(field, operator string, value any) (string, []any, error) {
	op := strings.ToUpper(strings.TrimSpace(operator))
	if op == "" {
		op = "="
	}

	switch op {
	case "=", "!=", "<>", ">", ">=", "<", "<=":
		return fmt.Sprintf("%s %s ?", field, op), []any{value}, nil

	case "LIKE", "ILIKE":
		return fmt.Sprintf("%s %s ?", field, op), []any{value}, nil

	case "IN", "NOT IN":
		return fmt.Sprintf("%s %s (?)", field, op), []any{value}, nil

	case "IS NULL", "NULL":
		return fmt.Sprintf("%s IS NULL", field), nil, nil

	case "IS NOT NULL", "NOT NULL":
		return fmt.Sprintf("%s IS NOT NULL", field), nil, nil

	case "BETWEEN", "NOT BETWEEN":
		valSlice, err := toSlice(value)
		if err != nil || len(valSlice) != 2 {
			return "", nil, fmt.Errorf("el operador '%s' para el campo '%s' requiere exactamente 2 valores", op, field)
		}
		return fmt.Sprintf("%s %s ? AND ?", field, op), []any{valSlice[0], valSlice[1]}, nil

	default:
		return "", nil, fmt.Errorf("operador de filtrado no soportado: '%s'", operator)
	}
}

// toSlice convierte un valor slice/array a []any.
func toSlice(val any) ([]any, error) {
	if val == nil {
		return nil, fmt.Errorf("valor nulo")
	}

	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("se esperaba un slice o array")
	}

	slice := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		slice[i] = v.Index(i).Interface()
	}
	return slice, nil
}