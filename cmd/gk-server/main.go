package main

// GophKeeper gRPC server.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/antonminaichev/gophkeeper/internal/config"
	"github.com/antonminaichev/gophkeeper/internal/server/handlers"
	"github.com/antonminaichev/gophkeeper/internal/server/middleware"
	"github.com/antonminaichev/gophkeeper/internal/server/storage/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.FromEnv()
	log.Printf("starting gRPC server on %s", cfg.GRPCAddr)

	ctx, cancel := signalContext()
	defer cancel()

	// DB pool
	pool, err := pgxpool.New(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	// Migrations with flag
	if cfg.RunMigrations {
		if err := postgres.RunMigrations(cfg.DBDSN); err != nil {
			log.Fatalf("migrations: %v", err)
		}
		log.Println("migrations: ok")
	}

	// JWT issuer
	priv, err := loadRSAPrivateKey(cfg.JWTPrivateKeyPath)
	if err != nil {
		log.Fatalf("load private key: %v", err)
	}
	issuer := auth.NewIssuer(priv, cfg.AccessTTL, cfg.RefreshTTL)

	// Repositories
	usersRepo := postgres.NewUserRepo(pool)
	itemsRepo := postgres.NewItemsRepo(pool)

	// gRPC server with auth middleware
	s := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthUnary(issuer.Public())),
	)

	// Services
	pb.RegisterAuthServiceServer(s, handlers.NewAuthServer(usersRepo, issuer))
	pb.RegisterVaultServiceServer(s, handlers.NewVaultServer(itemsRepo))

	// Health + reflection
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, hs)
	reflection.Register(s)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	// Serve in background, handle shutdown below.
	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve(lis) }()

	select {
	case <-ctx.Done():
		// Graceful shutdown.
		done := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
			log.Println("server stopped gracefully")
		case <-time.After(5 * time.Second):
			log.Println("graceful stop timed out; forcing stop")
			s.Stop()
		}
	case err := <-errCh:
		if err != nil {
			log.Fatalf("serve: %v", err)
		}
	}
}

// signalContext returns a context canceled on SIGINT/SIGTERM.
func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}

// loadRSAPrivateKey reads PKCS#1 or PKCS#8 PEM and returns *rsa.PrivateKey.
func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key in PKCS#8")
		}
		return rk, nil
	default:
		return nil, fmt.Errorf("unsupported PEM type: %s", block.Type)
	}
}
