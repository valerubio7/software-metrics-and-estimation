package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
)

func main() {
	config, err := api.LoadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping PostgreSQL: %v", err)
	}

	handler := api.NewHTTPHandler(postgres.NewPostgresProjectRepository(pool), api.NewProjectID)
	server := &http.Server{Addr: config.Address, Handler: handler}

	log.Printf("API listening on %s", config.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}
