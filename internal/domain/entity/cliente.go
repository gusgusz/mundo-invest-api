package entity

import (
	"errors"
	"regexp"
	"time"
)

// Prioridade representa o nível de prioridade de um cliente.
type Prioridade string

const (
	PrioridadeAlta   Prioridade = "prioridade_alta"
	PrioridadeNormal Prioridade = "prioridade_normal"
)

// Status representa o estado de atendimento do cliente no fluxo.
type Status string

const (
	StatusAguardandoAnalise Status = "Aguardando Análise"
	StatusProcessado        Status = "Processado"
)

// Cliente é a entidade central do domínio.
// Não possui dependências externas — apenas regras de negócio puras.
type Cliente struct {
	ID              uint       `json:"id"`
	Nome            string     `json:"cliente_nome"`
	Email           string     `json:"cliente_email"`
	TipoSolicitacao string     `json:"tipo_solicitacao"`
	ValorPatrimonio float64    `json:"valor_patrimonio"`
	Status          Status     `json:"status"`
	Prioridade      Prioridade `json:"prioridade,omitempty"`
	// EventID armazena o último event_id processado — garante idempotência do webhook.
	EventID   string    `json:"event_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// emailRegex valida o formato básico de um endereço de e-mail.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// NewCliente cria e valida uma nova instância de Cliente.
// Regra: status inicial é sempre "Aguardando Análise".
func NewCliente(nome, email, tipoSolicitacao string, valorPatrimonio float64) (*Cliente, error) {
	c := &Cliente{
		Nome:            nome,
		Email:           email,
		TipoSolicitacao: tipoSolicitacao,
		ValorPatrimonio: valorPatrimonio,
		Status:          StatusAguardandoAnalise,
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Validate verifica as invariantes da entidade.
func (c *Cliente) Validate() error {
	if c.Nome == "" {
		return errors.New("cliente_nome é obrigatório")
	}
	if c.Email == "" {
		return errors.New("cliente_email é obrigatório")
	}
	if !emailRegex.MatchString(c.Email) {
		return errors.New("cliente_email inválido")
	}
	if c.TipoSolicitacao == "" {
		return errors.New("tipo_solicitacao é obrigatório")
	}
	if c.ValorPatrimonio < 0 {
		return errors.New("valor_patrimonio não pode ser negativo")
	}
	return nil
}

// CalcularPrioridade aplica a regra de negócio de prioridade com base no patrimônio.
// Regra: patrimônio >= 200.000 → prioridade_alta; caso contrário → prioridade_normal.
func (c *Cliente) CalcularPrioridade() {
	if c.ValorPatrimonio >= 200_000 {
		c.Prioridade = PrioridadeAlta
	} else {
		c.Prioridade = PrioridadeNormal
	}
}

// Processar muda o status para "Processado", calcula a prioridade e registra o event_id.
// É chamado pelo UseCase ao processar o webhook.
func (c *Cliente) Processar(eventID string) {
	c.CalcularPrioridade()
	c.Status = StatusProcessado
	c.EventID = eventID
}
