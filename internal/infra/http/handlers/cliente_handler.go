package handlers

import (
	"github.com/gofiber/fiber/v2"
	domainUsecase "github.com/gusgusz/mundo-invest-api/internal/domain/usecase"
)

// ClienteHandler concentra os endpoints relacionados à entidade Cliente.
type ClienteHandler struct {
	uc domainUsecase.ClienteUseCase
}

// NewClienteHandler cria o handler injetando o caso de uso.
func NewClienteHandler(uc domainUsecase.ClienteUseCase) *ClienteHandler {
	return &ClienteHandler{uc: uc}
}

// CreateCliente godoc
// POST /clientes
// Valida o payload, persiste o cliente com status "Aguardando Análise" e
// loga a mutation GraphQL createCard do Pipefy no terminal.
func (h *ClienteHandler) CreateCliente(c *fiber.Ctx) error {
	var input domainUsecase.CreateClienteInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "payload inválido: " + err.Error(),
		})
	}

	output, err := h.uc.CreateCliente(input)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(output.Cliente)
}
