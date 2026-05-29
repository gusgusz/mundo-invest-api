package usecase

import (
	"context"
	"fmt"
	"log"

	"github.com/gusgusz/mundo-invest-api/internal/domain/apperror"
	"github.com/gusgusz/mundo-invest-api/internal/domain/entity"
	domainRepo "github.com/gusgusz/mundo-invest-api/internal/domain/repository"
	domainUsecase "github.com/gusgusz/mundo-invest-api/internal/domain/usecase"
)

// ClienteUseCaseImpl é a implementação concreta da interface ClienteUseCase.
// Orquestra as regras de domínio, persistência e integração com o Pipefy.
// Depende apenas de interfaces — nunca de implementações concretas.
type ClienteUseCaseImpl struct {
	repo      domainRepo.ClienteRepository
	pipefy    domainUsecase.PipefyService
	txManager domainRepo.TxManager
}

// Garante em tempo de compilação que *ClienteUseCaseImpl satisfaz domainUsecase.ClienteUseCase.
var _ domainUsecase.ClienteUseCase = (*ClienteUseCaseImpl)(nil)

// NewClienteUseCase cria uma nova instância do caso de uso com as dependências injetadas.
func NewClienteUseCase(
	repo domainRepo.ClienteRepository,
	pipefy domainUsecase.PipefyService,
	txManager domainRepo.TxManager,
) *ClienteUseCaseImpl {
	return &ClienteUseCaseImpl{repo: repo, pipefy: pipefy, txManager: txManager}
}

// CreateCliente executa o Fluxo 1:
//  1. Valida os dados de entrada e cria a entidade Cliente (status "Aguardando Análise").
//  2. Persiste o cliente no banco de dados.
//  3. Monta a mutation GraphQL `createCard` do Pipefy e loga o payload no terminal.
//
// Erros retornados:
//   - *apperror.AppError com CodeValidation para dados inválidos.
//   - *apperror.AppError com CodeConflict para e-mail já cadastrado.
//   - *apperror.AppError com CodeInternal para falhas de infraestrutura.
func (uc *ClienteUseCaseImpl) CreateCliente(input domainUsecase.CreateClienteInput) (*domainUsecase.CreateClienteOutput, error) {
	// ── 1. Criar e validar entidade ──────────────────────────────────────────────
	cliente, err := entity.NewCliente(
		input.Nome,
		input.Email,
		input.TipoSolicitacao,
		input.ValorPatrimonio,
	)
	if err != nil {
		return nil, apperror.Validation(fmt.Sprintf("dados inválidos: %s", err.Error()))
	}

	// ── 2. Persistir no banco ────────────────────────────────────────────────────
	// A unicidade do e-mail é garantida pelo índice único no banco.
	// Em caso de duplicata, o repositório retorna apperror.Conflict automaticamente.
	if err := uc.repo.Create(cliente); err != nil {
		return nil, err
	}

	// ── 3. Montar e logar mutation Pipefy (createCard) ───────────────────────────
	mutation := uc.pipefy.BuildCreateCardMutation(cliente)
	log.Printf("[PIPEFY] createCard payload:\n%s\n", mutation)

	return &domainUsecase.CreateClienteOutput{
		Cliente:        cliente,
		PipefyMutation: mutation,
	}, nil
}

// ProcessWebhookCardUpdated executa o Fluxo 2 dentro de uma transação atômica:
//  1. Idempotência — interrompe silenciosamente se o event_id já foi processado.
//  2. Busca o cliente pelo e-mail recebido no payload.
//  3. Aplica a regra de negócio de prioridade e muda o status para "Processado".
//  4. Persiste as alterações no banco.
//  5. Monta a mutation GraphQL `updateCardField` do Pipefy e loga no terminal.
//
// Todo o fluxo (check idempotência + update) roda dentro de uma única transação,
// garantindo que dois eventos concorrentes com o mesmo event_id não sejam ambos
// processados (TOCTOU race condition).
func (uc *ClienteUseCaseImpl) ProcessWebhookCardUpdated(input domainUsecase.WebhookCardUpdatedInput) error {
	return uc.txManager.RunInTx(context.Background(), func(txRepo domainRepo.ClienteRepository) error {
		// ── 1. Idempotência (atômica dentro da transação) ─────────────────────────
		existing, err := txRepo.FindByEventID(input.EventID)
		if err != nil {
			return err
		}
		if existing != nil {
			log.Printf("[WEBHOOK] event_id %q já processado — ignorando duplicata.", input.EventID)
			return nil
		}

		// ── 2. Buscar cliente pelo e-mail ────────────────────────────────────────
		cliente, err := txRepo.FindByEmail(input.ClienteEmail)
		if err != nil {
			return err
		}
		if cliente == nil {
			return apperror.NotFound("cliente não encontrado para o e-mail informado")
		}

		// ── 3. Aplicar regra de negócio (domínio puro) ───────────────────────────
		cliente.Processar(input.EventID)

		// ── 4. Persistir alterações ──────────────────────────────────────────────
		if err := txRepo.Update(cliente); err != nil {
			return err
		}

		// ── 5. Montar e logar mutation Pipefy (updateCardField) ──────────────────
		mutation := uc.pipefy.BuildUpdateCardFieldMutation(input.CardID, cliente.Status, cliente.Prioridade)
		log.Printf("[PIPEFY] updateCardField payload:\n%s\n", mutation)

		return nil
	})
}
