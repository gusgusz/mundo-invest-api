package usecase

import "github.com/gusgusz/mundo-invest-api/internal/domain/entity"

// PipefyService define a porta de saída responsável por montar os payloads
// GraphQL destinados ao Pipefy. A camada de UseCase depende desta interface —
// nunca da implementação concreta em infra/pipefy.
type PipefyService interface {
	// BuildCreateCardMutation monta a mutation GraphQL de criação de card,
	// seguindo a especificação oficial da API do Pipefy.
	// Retorna a string completa do payload (query + variables) pronta para envio.
	BuildCreateCardMutation(cliente *entity.Cliente) string

	// BuildUpdateCardFieldMutation monta a mutation GraphQL de atualização de campos
	// de um card existente, conforme a spec do Pipefy (updateCardField).
	// Recebe o card_id, o novo status e a prioridade calculada.
	BuildUpdateCardFieldMutation(cardID string, status entity.Status, prioridade entity.Prioridade) string
}
