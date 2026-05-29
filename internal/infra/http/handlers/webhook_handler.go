package handlers

import (
	"github.com/gofiber/fiber/v2"
	domainUsecase "github.com/gusgusz/mundo-invest-api/internal/domain/usecase"
)

// WebhookHandler concentra os endpoints de webhooks externos.
type WebhookHandler struct {
	uc domainUsecase.ClienteUseCase
}

// NewWebhookHandler cria o handler injetando o caso de uso.
func NewWebhookHandler(uc domainUsecase.ClienteUseCase) *WebhookHandler {
	return &WebhookHandler{uc: uc}
}

// CardUpdated godoc
// POST /webhooks/pipefy/card-updated
// Simula o recebimento de um evento do Pipefy quando um card é atualizado.
// Garante idempotência pelo event_id, aplica regra de prioridade e loga
// a mutation GraphQL updateCardField no terminal.
func (h *WebhookHandler) CardUpdated(c *fiber.Ctx) error {
	var input domainUsecase.WebhookCardUpdatedInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "payload inválido: " + err.Error(),
		})
	}

	if err := h.uc.ProcessWebhookCardUpdated(input); err != nil {
		return errorResponse(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "ok"})
}
