package server

import (
	"github.com/gofiber/fiber/v2"
	domainUsecase "github.com/gusgusz/mundo-invest-api/internal/domain/usecase"
	"github.com/gusgusz/mundo-invest-api/internal/infra/http/handlers"
)

// SetupRoutes registra todas as rotas da aplicação no app Fiber.
func SetupRoutes(app *fiber.App, uc domainUsecase.ClienteUseCase) {
	clienteHandler := handlers.NewClienteHandler(uc)
	webhookHandler := handlers.NewWebhookHandler(uc)

	// Fluxo 1: Criação de cliente
	app.Post("/clientes", clienteHandler.CreateCliente)

	// Fluxo 2: Webhook de atualização de card do Pipefy
	app.Post("/webhooks/pipefy/card-updated", webhookHandler.CardUpdated)
}
