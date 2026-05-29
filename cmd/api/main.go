package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/gusgusz/mundo-invest-api/internal/infra/database"
	dbRepo "github.com/gusgusz/mundo-invest-api/internal/infra/database/repository"
	server "github.com/gusgusz/mundo-invest-api/internal/infra/http"
	"github.com/gusgusz/mundo-invest-api/internal/infra/pipefy"
	"github.com/gusgusz/mundo-invest-api/internal/usecase"
)

func main() {
	// Carrega variáveis do .env (ignora erro caso rode via variáveis do sistema, ex: Docker).
	if err := godotenv.Load(); err != nil {
		log.Println("[CONFIG] Arquivo .env não encontrado — usando variáveis do ambiente do sistema.")
	}

	// ── Banco de dados ────────────────────────────────────────────────────────
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("[DB] Falha ao conectar: %v", err)
	}

	// ── Injeção de dependências (Composition Root) ────────────────────────────
	repo := dbRepo.NewClienteRepository(db)
	txManager := database.NewTxManager(db)
	pipefyClient := pipefy.NewClient(os.Getenv("PIPEFY_PIPE_ID"))
	uc := usecase.NewClienteUseCase(repo, pipefyClient, txManager)

	// ── HTTP (Fiber) ──────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName: "Mundo Invest API v1.0",
	})
	app.Use(logger.New())
	app.Use(recover.New())

	server.SetupRoutes(app, uc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("[SERVER] Iniciando na porta :%s", port)
	log.Fatal(app.Listen(":" + port))
}
