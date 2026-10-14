# sqlmanager

**sqlmanager** es una plantilla y microservicio de infraestructura en Go diseñado para actuar como puente de persistencia entre microservicios y una base de datos relacional (PostgreSQL o MySQL/MariaDB) a través de gRPC.

Provee un andamiaje robusto y desacoplado que resuelve de fábrica el 90% de la lógica pesada de infraestructura: auditoría automática, repositorios genéricos con transacciones atómicas, calibración de conexiones, consultas dinámicas con paginación segura, monitoreo de salud y apagado elegante en contenedores Docker.

---

## 🚀 Características Implementadas

### 1. Repositorio Genérico (`BaseRepository[T]`)
Implementa un CRUD estandarizado sobre cualquier entidad mediante genéricos de Go (`[T any]`), integrando validación estricta de `TableName()` y manejo contextualizado de errores.

| Método | Entrada | Salida | Descripción |
| :--- | :--- | :--- | :--- |
| **`Create`** | `(ctx, entity *T)` | `(id, error)` | Inserta el registro, asigna `CreatedBy` y guarda el evento `INSERT` en la tabla `auditoria`. |
| **`Read`** | `(ctx, id string)` | `(*T, error)` | Búsqueda directa `O(1)` por clave primaria respetando el borrado lógico. |
| **`FindOne`** | `(ctx, filters []Filter)` | `(*T, error)` | Busca el primer registro que cumpla con criterios dinámicos (`LIMIT 1`). |
| **`Exists`** | `(ctx, id string)` | `(bool, error)` | Comprobación ultraliviana de existencia (`SELECT 1 ... LIMIT 1`) sin hidratar structs en memoria. |
| **`List`** | `(ctx, params ListParams)` | `([]T, *PaginationMetadata, error)` | Listado paginado con metadata fija, ordenamiento seguro y filtros anidados. |
| **`Update`** | `(ctx, id string, data *T)` | `(id, error)` | Calcula el delta de cambios, asigna `UpdatedBy` y registra el evento `UPDATE` en `auditoria`. |
| **`SoftDelete`** | `(ctx, id string)` | `(id, error)` | Asigna `deleted_by`, aplica borrado lógico en GORM y registra `DELETE` en `auditoria`. |
| **`Restore`** | `(ctx, id string)` | `(id, error)` | Reactiva un registro borrado lógicamente, limpia `deleted_at/by` y registra `RESTORE`. |
| **`HardDelete`** | `(ctx, id string)` | `(mensaje, error)` | Borrado físico para **Sagas / Rollbacks**. Valida filas afectadas y purga el historial en `auditoria`. |

---

### 2. Motor de Filtros y Paginación Dinámica (`select_builder.go`)
- **Sintaxis infija hacia adelante (*Next Logic*):** Cada filtro define mediante su campo `Logic` cómo se une con el siguiente elemento a su mismo nivel (`AND` u `OR`). El último elemento de un nivel o grupo se deja vacío.
- **Grupos anidados con paréntesis:** Permite anidar condiciones complejas mediante `Group: []Filter` (ej: `A AND (B OR C)`).
- **Operadores soportados:** `=`, `!=`, `<>`, `>`, `>=`, `<`, `<=`, `LIKE`, `ILIKE`, `IN`, `NOT IN`, `BETWEEN`, `NOT BETWEEN`, `IS NULL`, `IS NOT NULL`.
- **Protección anti-inyección SQL:** Valida que los nombres de columnas contengan exclusivamente caracteres seguros (`^[a-zA-Z0-9_.]+$`).
- **Normalización de paginación:** Aplica límites seguros por defecto (página 1, tamaño 20, máximo 100).

---

### 3. Trazabilidad y Auditoría Automática Transaccional
- **Tabla de sistema obligatoria (`auditoria`):** Toda mutación (`INSERT`, `UPDATE`, `DELETE`, `RESTORE`) se ejecuta dentro de una transacción atómica junto con el registro en la tabla `auditoria`. Si la auditoría falla, se revierte la mutación.
- **Formato uniforme de diff:**
  ```json
  {
    "campo": {
      "anterior": any,
      "actual": any
    }
  }
  ```
- **Interceptor gRPC (`auditoria_interceptor.go`):** Extrae automáticamente de los headers HTTP/2 las cabeceras `x-autor-id`, `x-autor-ip` y `x-user-agent`, inyectándolas en el `context.Context` sin ensuciar los mensajes Protobuf.

---

### 4. Infraestructura y Operaciones
- **Calibración del Pool de Conexiones (`connection.go`):** Ajuste de `MaxOpenConns`, `MaxIdleConns` y `ConnMaxLifetime` para proteger la base de datos de saturación y evitar conexiones caídas en Docker.
- **Soporte Multi-Driver:** Compatible con **PostgreSQL** y **MySQL / MariaDB** mediante variable de entorno `DB_DRIVER`.
- **Graceful Shutdown:** Atrapa señales del sistema operativo (`SIGINT`, `SIGTERM`) enviadas por Docker para permitir que las peticiones en vuelo terminen (`server.GracefulStop()`), cerrando el pool de BD limpiamente.
- **gRPC Health Check:** Implementa el protocolo oficial `grpc.health.v1` con monitoreo nativo mediante `grpc-health-probe` en Docker Compose.
- **Server Reflection:** Permite que clientes como Postman o `grpcurl` auto-descubran los contratos gRPC. Protegido para activarse **únicamente en modo desarrollo** (`APP_ENV=dev`).
- **Migraciones y Seeders:** Estructura modularizada en `internal/database/migrations.go` y `internal/database/seeders.go`.

---

## 🛠️ Requisitos Previos

1. **Docker y Docker Compose** instalados en el sistema anfitrión.
2. **Red externa de Docker** compartida con el resto de los microservicios:
   ```bash
   docker network create template-network
   ```
3. **Contenedor de Base de Datos:** Un contenedor de PostgreSQL o MySQL/MariaDB corriendo en la misma red de Docker.

---

## 📖 Tutorial: Puesta en Marcha tras Clonar el Repositorio

### Paso 1: Configurar los archivos de entorno (`.env`)

El proyecto utiliza dos archivos de entorno:

1. **Configuración de Docker (Raíz):**
   ```bash
   cp .env.example .env
   ```
   Edita el archivo `.env` en la raíz para definir el prefijo del proyecto:
   ```env
   PROJECT_NAME=mi-servicio
   ```

2. **Configuración de la Aplicación Go (`volume/app/src/`):**
   ```bash
   cp volume/app/src/.env.example volume/app/src/.env
   ```
   Configura las credenciales de tu base de datos:
   ```env
   APP_ENV=dev
   GRPC_HOST=0.0.0.0
   GRPC_PORT=50051

   DB_DRIVER=postgres # Opciones: postgres | mysql
   DB_HOST=tu-contenedor-db
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=secret
   DB_NAME=mi_base_de_datos

   # Pool de conexiones (opcional)
   # DB_MAX_OPEN_CONNS=25
   # DB_MAX_IDLE_CONNS=10
   # DB_CONN_MAX_LIFETIME=15
   ```

---

### Paso 2: Iniciar el contenedor en Desarrollo

Levanta los contenedores en segundo plano:
```bash
docker compose up -d
```
> En modo desarrollo, el contenedor utiliza **Air** para detectar cambios en los archivos `.go` y recompilar automáticamente en caliente (*hot-reload*).

Para revisar los logs en tiempo real:
```bash
docker compose logs -f app
```

Para verificar que el Health Check esté activo y saludable:
```bash
docker compose ps
# La columna STATUS indicará: Up ... (healthy)
```

---

### Paso 3: Acceder al contenedor por terminal

Para ejecutar comandos dentro del contenedor de la aplicación:
```bash
docker compose exec app sh
```

---

### Paso 4: Despliegue en Producción

Para compilar un binario estático optimizado en Alpine y levantar en modo producción:
```bash
docker compose -f docker-compose.prod.yml up -d --build
```

---

## 🧩 Guía de Extensión: Cómo Implementar Modelos y Servicios de Negocio

Cuando clones esta plantilla para construir un microservicio concreto, sigue estos pasos:

### 1. Crear el Modelo de Negocio
Crea tu estructura en `volume/app/src/internal/models/<nombre>.go`. Debe incrustar obligatoriamente `models.Base` e implementar la interfaz `TableName() string`:

```go
package models

type Producto struct {
    Base // Incluye ID (UUIDv7), CreatedAt, UpdatedAt, DeletedAt, CreatedBy, UpdatedBy, DeletedBy

    Nombre string  `gorm:"type:varchar(100);not null" json:"nombre"`
    Precio float64 `gorm:"type:decimal(10,2);not null" json:"precio"`
    Stock  int     `gorm:"not null;default:0" json:"stock"`
}

// TableName define el nombre exacto de la tabla (requerido por BaseRepository)
func (Producto) TableName() string {
    return "productos"
}
```

---

### 2. Registrar en Migraciones
Agrega el nuevo modelo en `volume/app/src/internal/database/migrations.go`:

```go
err := db.AutoMigrate(
    &models.Auditoria{}, // Modelo obligatorio del sistema
    &models.Producto{},  // Tu nuevo modelo de negocio
)
```

---

### 3. Usar el Repositorio Base
Puedes instanciar directamente `BaseRepository` para tu entidad:

```go
productoRepo := repositories.NewBaseRepository[models.Producto](db)

// Crear con auditoría automática
id, err := productoRepo.Create(ctx, &models.Producto{
    Nombre: "Notebook",
    Precio: 1250.50,
    Stock:  10,
})

// Listar con paginación y filtros dinámicos
items, meta, err := productoRepo.List(ctx, database.ListParams{
    Page:     1,
    PageSize: 10,
    OrderBy:  "precio",
    OrderDir: "DESC",
    Filters: []database.Filter{
        {Field: "precio", Operator: ">", Value: 500, Logic: "AND"},
        {Field: "stock", Operator: ">=", Value: 1},
    },
})
```

---

### 4. Definir y Exponer Contratos Protobuf (gRPC)
1. Define tus contratos `.proto` en un directorio dedicado (ej: `volume/app/src/proto/producto.proto`).
2. Compila los stubs con `protoc`.
3. Implementa el servidor gRPC conectando las llamadas RPC con `productoRepo`.
4. Registra tu servicio en `main.go`:
   ```go
   pb.RegisterProductoServiceServer(server, productoHandler)
   ```
