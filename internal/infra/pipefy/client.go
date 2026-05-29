package pipefy

import (
	"fmt"

	"github.com/gusgusz/mundo-invest-api/internal/domain/entity"
	domainUsecase "github.com/gusgusz/mundo-invest-api/internal/domain/usecase"
)

// Client implementa a interface domainUsecase.PipefyService.
// Sua única responsabilidade é montar as strings GraphQL exatas da API do Pipefy.
// Nenhuma requisição HTTP real é feita — o payload é apenas estruturado e logado.
type Client struct {
	// PipeID é o identificador do pipe (quadro) no Pipefy onde os cards serão criados.
	// Em produção este valor viria de variável de ambiente.
	PipeID string
}

// Garante em tempo de compilação que *Client satisfaz a interface PipefyService.
var _ domainUsecase.PipefyService = (*Client)(nil)

// NewClient cria uma instância do cliente Pipefy com o pipe_id configurado.
func NewClient(pipeID string) *Client {
	return &Client{PipeID: pipeID}
}

// BuildCreateCardMutation monta a mutation GraphQL de criação de card conforme
// a documentação oficial do Pipefy:
// https://developers.pipefy.com/reference/mutations-cards
//
// Mutation: createCard
// A mutation recebe um objeto `CreateCardInput` com:
//   - pipe_id       → identificador do pipe (quadro)
//   - title         → título do card (usamos o nome do cliente)
//   - fields_attributes → lista de campos customizados do pipe com field_id e field_value
//
// O retorno inclui o id e o title do card criado.
func (c *Client) BuildCreateCardMutation(cliente *entity.Cliente) string {
	// ─── Query GraphQL ───────────────────────────────────────────────────────────
	// Segue exatamente o schema da API do Pipefy. A mutation `createCard` aceita
	// `CreateCardInput!` e retorna um objeto `CreateCardPayload` com `card { id title }`.
	mutation := `mutation createCard($input: CreateCardInput!) {
  createCard(input: $input) {
    card {
      id
      title
    }
  }
}`

	// ─── Variáveis ───────────────────────────────────────────────────────────────
	// `fields_attributes` mapeia os campos do pipe; os field_ids correspondem
	// ao slug de cada campo customizado cadastrado no pipe de destino.
	//
	// Formato de field_value conforme documentação oficial do Pipefy
	// (developers.pipefy.com/reference — "Create a card with the required fields fulfilled"):
	//   - Campos de texto/email/número → string simples: "valor"
	//   - Campos do tipo label_select  → array de IDs:   ["id_do_label"]
	//
	// Os três campos abaixo são do tipo texto/email/número, portanto usam string simples.
	variables := fmt.Sprintf(`{
  "input": {
    "pipe_id": "%s",
    "title": "%s",
    "fields_attributes": [
      { "field_id": "cliente_email",    "field_value": "%s" },
      { "field_id": "tipo_solicitacao", "field_value": "%s" },
      { "field_id": "valor_patrimonio", "field_value": "%.2f" }
    ]
  }
}`,
		c.PipeID,
		cliente.Nome,
		cliente.Email,
		cliente.TipoSolicitacao,
		cliente.ValorPatrimonio,
	)

	return buildPayload(mutation, variables)
}

// BuildUpdateCardFieldMutation monta a mutation GraphQL de atualização de campo
// de um card conforme a documentação oficial do Pipefy:
// https://developers.pipefy.com/reference/mutations-cards
//
// Mutation: updateCardField
// A mutation recebe um objeto `UpdateCardFieldInput` com:
//   - card_id   → id do card a ser atualizado
//   - field_id  → slug do campo a ser alterado
//   - new_value → novo valor do campo (string)
//
// Para atualizar múltiplos campos (status + prioridade) são geradas duas chamadas
// aninhadas usando aliases GraphQL, o que é válido e idiomático na API do Pipefy.
func (c *Client) BuildUpdateCardFieldMutation(
	cardID string,
	status entity.Status,
	prioridade entity.Prioridade,
) string {
	// ─── Query GraphQL com aliases ───────────────────────────────────────────────
	// Aliases (`updateStatus`, `updatePrioridade`) permitem enviar múltiplas
	// execuções de updateCardField em uma única requisição GraphQL, conforme
	// previsto pela especificação GraphQL e pelo schema do Pipefy.
	mutation := `mutation updateCardFields(
  $inputStatus:    UpdateCardFieldInput!,
  $inputPrioridade: UpdateCardFieldInput!
) {
  updateStatus: updateCardField(input: $inputStatus) {
    card { id }
    success
  }
  updatePrioridade: updateCardField(input: $inputPrioridade) {
    card { id }
    success
  }
}`

	// ─── Variáveis ───────────────────────────────────────────────────────────────
	// `field_id` corresponde ao slug dos campos customizados do pipe.
	// `new_value` é string simples para campos de texto/seleção (não label_select).
	// Caso o campo fosse label_select, new_value seria o ID numérico do label.
	variables := fmt.Sprintf(`{
  "inputStatus": {
    "card_id":   "%s",
    "field_id":  "status_cliente",
    "new_value": "%s"
  },
  "inputPrioridade": {
    "card_id":   "%s",
    "field_id":  "prioridade",
    "new_value": "%s"
  }
}`,
		cardID, string(status),
		cardID, string(prioridade),
	)

	return buildPayload(mutation, variables)
}

// buildPayload formata o payload final no padrão de envio HTTP da API GraphQL do Pipefy:
// um JSON com os campos "query" e "variables" separados.
func buildPayload(query, variables string) string {
	return fmt.Sprintf("{ \"query\": %q, \"variables\": %s }", query, variables)
}
