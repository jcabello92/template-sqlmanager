package main

import (
	"fmt"
	"log"
	"net"

	"template-sqlmanager/internal/config"
	"template-sqlmanager/internal/database"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func main() {
	cfg := config.LoadEnv()

	db := database.Connect(cfg)
	database.RunMigrations(db)
	database.RunSeeders(db)

	_ = proto.Marshal

	// De aquí hacia abajo hay que estandarizar...
	address := fmt.Sprintf("%s:%s", cfg.GRPCHost, cfg.GRPCPort)
	
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error abriendo el puerto %s: %v", address, err)
	}
	server := grpc.NewServer()
	
	log.Printf("Servidor gRPC iniciado en %s (Modo: %s)\n", address, cfg.AppEnv)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Error en el servidor: %v", err)
	}
}