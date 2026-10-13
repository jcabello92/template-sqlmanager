# Plantilla Docker - Go (Backend)

Plantilla minimalista y multipropósito para Go en Docker. No incluye frameworks, ORMs ni transportes predefinidos.

## Requisitos previos

Crear la red externa de Docker definida en `.env` (si aún no existe):

```bash
docker network create template-network
```

## Uso en desarrollo

1. Iniciar los contenedores:
   ```bash
   docker compose up -d
   ```

2. Inicializar el módulo de Go (si aún no está creado):
   ```bash
   docker compose exec app go mod init <nombre-del-modulo>
   ```

3. Crear el archivo `main.go` en `./volume/app/src/`. El contenedor detectará `go.mod` y activará el hot-reload mediante **Air**.

## Uso en producción

Compilar y levantar en modo producción (binario estático optimizado en Alpine):

```bash
docker compose -f docker-compose.prod.yml up -d --build
```
