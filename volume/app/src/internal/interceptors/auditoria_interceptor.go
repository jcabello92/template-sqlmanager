package interceptors

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const auditContextKey contextKey = "audit_data"

// AuditData encapsula la información de auditoría extraída de las cabeceras gRPC.
type AuditData struct {
	AutorID   *uuid.UUID
	AutorIP   *string
	UserAgent *string
}

// AuditoriaInterceptor es un middleware unario para gRPC que extrae los metadatos de auditoría
// (x-autor-id, x-autor-ip, x-user-agent) y los inyecta en el context.Context.
func AuditoriaInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		audit := extractAuditData(ctx)
		newCtx := context.WithValue(ctx, auditContextKey, audit)
		return handler(newCtx, req)
	}
}

// extractAuditData lee los headers/metadata de gRPC y construye la estructura AuditData.
func extractAuditData(ctx context.Context) *AuditData {
	audit := &AuditData{}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return audit
	}

	// Extraer x-autor-id
	if vals := md.Get("x-autor-id"); len(vals) > 0 && strings.TrimSpace(vals[0]) != "" {
		if parsedUUID, err := uuid.Parse(strings.TrimSpace(vals[0])); err == nil {
			audit.AutorID = &parsedUUID
		}
	}

	// Extraer x-autor-ip
	if vals := md.Get("x-autor-ip"); len(vals) > 0 && strings.TrimSpace(vals[0]) != "" {
		ip := strings.TrimSpace(vals[0])
		audit.AutorIP = &ip
	}

	// Extraer x-user-agent
	if vals := md.Get("x-user-agent"); len(vals) > 0 && strings.TrimSpace(vals[0]) != "" {
		ua := strings.TrimSpace(vals[0])
		audit.UserAgent = &ua
	}

	return audit
}

// GetAuditFromContext obtiene la información de auditoría del contexto de forma segura.
func GetAuditFromContext(ctx context.Context) *AuditData {
	if ctx == nil {
		return &AuditData{}
	}
	if audit, ok := ctx.Value(auditContextKey).(*AuditData); ok && audit != nil {
		return audit
	}
	return &AuditData{}
}

// WithAuditContext inyecta datos de auditoría en un contexto (útil para pruebas unitarias o llamadas internas).
func WithAuditContext(ctx context.Context, audit *AuditData) context.Context {
	if audit == nil {
		audit = &AuditData{}
	}
	return context.WithValue(ctx, auditContextKey, audit)
}
