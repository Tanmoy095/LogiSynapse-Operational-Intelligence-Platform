package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/app/commands"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/config"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/infra/postgres"
	authcrypto "github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/ports/crypto"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/transport/authapi"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/transport/grpcjson"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	grpcjson.Register()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if cfg.AllowAutoMigrate {
		if err := runMigrations(ctx, db, "db/migrations"); err != nil {
			log.Fatalf("run migrations: %v", err)
		}
	}

	store := postgres.NewPostgresStore(db)
	hasher := authcrypto.NewArgon2Hasher(nil)
	signer := authcrypto.NewHMACTokenSigner(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL)

	register := commands.NewRegisterUserHandler(store, hasher, store)
	login := commands.NewLoginUserHandler(store, store, hasher, signer)
	createTenant := commands.NewCreateTenantCmdByPlatform(store, store, store)
	addMembership := commands.NewAddMembershipCmd(store, store, store, store)
	authServer := authapi.NewServer(register, login, createTenant, addMembership, signer, store, store, store, store)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen on %s: %v", cfg.GRPCAddr, err)
	}

	server := grpc.NewServer(grpc.ForceServerCodec(grpcjson.Codec{}))
	authapi.RegisterGRPCServer(server, authServer)
	log.Printf("authentication-service listening on %s", cfg.GRPCAddr)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("serve auth gRPC: %v", err)
	}
}

func runMigrations(ctx context.Context, db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migration dir %s: %w", dir, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		sqlText, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", path, err)
		}
		if _, err := db.ExecContext(ctx, string(sqlText)); err != nil {
			return fmt.Errorf("execute migration %s: %w", path, err)
		}
	}
	return nil
}
