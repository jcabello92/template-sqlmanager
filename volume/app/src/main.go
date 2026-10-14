package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"template-sqlmanager/internal/config"
	"template-sqlmanager/internal/database"
	"template-sqlmanager/internal/interceptors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
)

func main() {
	cfg := config.LoadEnv()

	// 1. Conexión y preparación de base de datos
	db := database.Connect(cfg)
	database.RunMigrations(db)
	database.RunSeeders(db)

	_ = proto.Marshal

	// 2. Preparar el listener gRPC
	address := fmt.Sprintf("%s:%s", cfg.GRPCHost, cfg.GRPCPort)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error abriendo el puerto %s: %v", address, err)
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.AuditoriaInterceptor()),
	)

	// Registrar servicio estándar de Health Check (grpc.health.v1)
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Habilitar Reflection solo en desarrollo para herramientas como Postman / grpcurl
	if cfg.AppEnv == "dev" {
		reflection.Register(server)
		log.Println("gRPC Server Reflection habilitado (Modo desarrollo).")
	}

	// 3. Iniciar el servidor gRPC en segundo plano (no bloqueante)
	go func() {
		log.Printf("Servidor gRPC iniciado en %s (Modo: %s)\n", address, cfg.AppEnv)
		if err := server.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("Error crítico en el servidor gRPC: %v", err)
		}
	}()

	// 4. Capturar señales de apagado del sistema / Docker (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Señal de apagado recibida. Deteniendo servicios de forma segura...")

	// Marcar Health Check como NOT_SERVING para que los balanceadores dejen de enviar tráfico de inmediato
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// 5. Apagado elegante de gRPC con timeout máximo de 10 segundos
	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("Servidor gRPC detenido correctamente.")
	case <-time.After(10 * time.Second):
		log.Println("Tiempo de espera agotado. Forzando detención del servidor gRPC...")
		server.Stop()
	}

	// 6. Cerrar el pool de conexiones de la base de datos
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("Error al cerrar conexiones de la base de datos: %v\n", err)
		} else {
			log.Println("Conexiones de base de datos cerradas exitosamente.")
		}
	}

	log.Println("Microservicio finalizado correctamente.")
}