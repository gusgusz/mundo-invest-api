package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/gusgusz/mundo-invest-api/internal/domain/apperror"
)

// errorResponse mapeia um *apperror.AppError para o status HTTP correto e retorna
// um JSON padronizado com o campo "error". Se o erro não for um AppError, retorna 500.
//
// Mapeamento:
//
//	CodeNotFound   → 404
//	CodeConflict   → 409
//	CodeValidation → 422
//	CodeInternal   → 500
func errorResponse(c *fiber.Ctx, err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		status := fiber.StatusInternalServerError
		switch appErr.Code {
		case apperror.CodeNotFound:
			status = fiber.StatusNotFound
		case apperror.CodeConflict:
			status = fiber.StatusConflict
		case apperror.CodeValidation:
			status = fiber.StatusUnprocessableEntity
		}
		return c.Status(status).JSON(fiber.Map{
			"error": appErr.Message,
			"code":  string(appErr.Code),
		})
	}
	// Fallback para erros não tipados — não vazar detalhes internos.
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "erro interno do servidor",
		"code":  string(apperror.CodeInternal),
	})
}
