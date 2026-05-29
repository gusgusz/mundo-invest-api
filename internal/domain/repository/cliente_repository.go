package repository

import "github.com/gusgusz/mundo-invest-api/internal/domain/entity"

// ClienteRepository define o contrato de persistência da entidade Cliente.
// Qualquer implementação concreta (GORM, in-memory, etc.) deve satisfazer esta interface.
type ClienteRepository interface {
	// Create persiste um novo cliente no banco de dados.
	Create(cliente *entity.Cliente) error

	// FindByEmail busca um cliente pelo seu endereço de e-mail.
	// Retorna nil se não encontrado.
	FindByEmail(email string) (*entity.Cliente, error)

	// FindByEventID verifica se já existe um registro com o event_id informado.
	// Utilizado para garantir idempotência no processamento do webhook.
	FindByEventID(eventID string) (*entity.Cliente, error)

	// Update persiste as alterações de um cliente existente.
	Update(cliente *entity.Cliente) error
}
