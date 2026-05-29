package usecase

import "github.com/gusgusz/mundo-invest-api/internal/domain/entity"

// CreateClienteInput contém os dados necessários para criar um novo cliente.
type CreateClienteInput struct {
	Nome            string  `json:"cliente_nome"`
	Email           string  `json:"cliente_email"`
	TipoSolicitacao string  `json:"tipo_solicitacao"`
	ValorPatrimonio float64 `json:"valor_patrimonio"`
}

// CreateClienteOutput é retornado após a criação bem-sucedida de um cliente.
type CreateClienteOutput struct {
	Cliente        *entity.Cliente `json:"cliente"`
	PipefyMutation string          `json:"pipefy_mutation"`
}

// WebhookCardUpdatedInput contém o payload recebido no endpoint de webhook.
type WebhookCardUpdatedInput struct {
	EventID      string `json:"event_id"`
	CardID       string `json:"card_id"`
	ClienteEmail string `json:"cliente_email"`
	Timestamp    string `json:"timestamp"`
}

// ClienteUseCase define os contratos dos casos de uso da entidade Cliente.
// A camada de handlers (HTTP) depende apenas desta interface, nunca da implementação.
type ClienteUseCase interface {
	// CreateCliente valida, persiste o cliente e retorna a mutation GraphQL montada.
	CreateCliente(input CreateClienteInput) (*CreateClienteOutput, error)

	// ProcessWebhookCardUpdated processa o evento de atualização de card do Pipefy,
	// garantindo idempotência pelo event_id e aplicando a regra de prioridade.
	ProcessWebhookCardUpdated(input WebhookCardUpdatedInput) error
}
