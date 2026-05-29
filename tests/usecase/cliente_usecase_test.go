package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gusgusz/mundo-invest-api/internal/domain/apperror"
	"github.com/gusgusz/mundo-invest-api/internal/domain/entity"
	domainRepo "github.com/gusgusz/mundo-invest-api/internal/domain/repository"
	domainUsecase "github.com/gusgusz/mundo-invest-api/internal/domain/usecase"
	"github.com/gusgusz/mundo-invest-api/internal/usecase"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

// mockClienteRepo implementa domainRepo.ClienteRepository em memória.
type mockClienteRepo struct {
	clientes  []*entity.Cliente
	createErr error
	updateErr error
}

var _ domainRepo.ClienteRepository = (*mockClienteRepo)(nil)

func (m *mockClienteRepo) Create(c *entity.Cliente) error {
	if m.createErr != nil {
		return m.createErr
	}
	c.ID = uint(len(m.clientes) + 1)
	m.clientes = append(m.clientes, c)
	return nil
}

func (m *mockClienteRepo) FindByEmail(email string) (*entity.Cliente, error) {
	for _, c := range m.clientes {
		if c.Email == email {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockClienteRepo) FindByEventID(eventID string) (*entity.Cliente, error) {
	if eventID == "" {
		return nil, nil
	}
	for _, c := range m.clientes {
		if c.EventID == eventID {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockClienteRepo) Update(c *entity.Cliente) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, existing := range m.clientes {
		if existing.Email == c.Email {
			m.clientes[i] = c
			return nil
		}
	}
	return errors.New("cliente nao encontrado para update")
}

// mockPipefyService implementa domainUsecase.PipefyService retornando strings fixas.
type mockPipefyService struct{}

var _ domainUsecase.PipefyService = (*mockPipefyService)(nil)

func (m *mockPipefyService) BuildCreateCardMutation(c *entity.Cliente) string {
	return `{"query":"mutation createCard(...)","variables":{}}`
}

func (m *mockPipefyService) BuildUpdateCardFieldMutation(
	cardID string, status entity.Status, prioridade entity.Prioridade,
) string {
	return `{"query":"mutation updateCardField(...)","variables":{}}`
}

// mockTxManager implementa domainRepo.TxManager sem banco real.
// Executa fn diretamente com o repositório em memória — correto para testes unitários.
type mockTxManager struct {
	repo domainRepo.ClienteRepository
}

var _ domainRepo.TxManager = (*mockTxManager)(nil)

func (m *mockTxManager) RunInTx(_ context.Context, fn func(domainRepo.ClienteRepository) error) error {
	return fn(m.repo)
}

// ── Helper ─────────────────────────────────────────────────────────────────────

func newUseCase(repo domainRepo.ClienteRepository) domainUsecase.ClienteUseCase {
	txm := &mockTxManager{repo: repo}
	return usecase.NewClienteUseCase(repo, &mockPipefyService{}, txm)
}

// ── Testes ─────────────────────────────────────────────────────────────────────

// Teste 1 — Criacao de cliente com payload valido e salvamento no banco.
func TestCreateCliente_Success(t *testing.T) {
	repo := &mockClienteRepo{}
	uc := newUseCase(repo)

	input := domainUsecase.CreateClienteInput{
		Nome:            "Joao Silva",
		Email:           "joao.silva@example.com",
		TipoSolicitacao: "Atualizacao cadastral",
		ValorPatrimonio: 250_000,
	}

	output, err := uc.CreateCliente(input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, "Joao Silva", output.Cliente.Nome)
	assert.Equal(t, entity.StatusAguardandoAnalise, output.Cliente.Status)
	assert.NotEmpty(t, output.PipefyMutation)
	assert.Len(t, repo.clientes, 1)
}

// Teste 1b — Criacao com e-mail invalido deve retornar erro de validacao.
func TestCreateCliente_EmailInvalido(t *testing.T) {
	repo := &mockClienteRepo{}
	uc := newUseCase(repo)

	_, err := uc.CreateCliente(domainUsecase.CreateClienteInput{
		Nome:            "Fulano",
		Email:           "email-invalido",
		TipoSolicitacao: "Consulta",
		ValorPatrimonio: 100_000,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "email")
	assert.Empty(t, repo.clientes)
}

// Teste 2a — Webhook aplica prioridade_alta quando patrimonio >= 200.000.
func TestProcessWebhook_PrioridadeAlta(t *testing.T) {
	repo := &mockClienteRepo{}
	uc := newUseCase(repo)

	_, err := uc.CreateCliente(domainUsecase.CreateClienteInput{
		Nome:            "Maria Alta",
		Email:           "maria@example.com",
		TipoSolicitacao: "Aplicacao",
		ValorPatrimonio: 200_000,
	})
	require.NoError(t, err)

	err = uc.ProcessWebhookCardUpdated(domainUsecase.WebhookCardUpdatedInput{
		EventID:      "evt_001",
		CardID:       "card_001",
		ClienteEmail: "maria@example.com",
		Timestamp:    "2026-05-18T12:00:00Z",
	})

	require.NoError(t, err)
	assert.Equal(t, entity.StatusProcessado, repo.clientes[0].Status)
	assert.Equal(t, entity.PrioridadeAlta, repo.clientes[0].Prioridade)
}

// Teste 2b — Webhook aplica prioridade_normal quando patrimonio < 200.000.
func TestProcessWebhook_PrioridadeNormal(t *testing.T) {
	repo := &mockClienteRepo{}
	uc := newUseCase(repo)

	_, err := uc.CreateCliente(domainUsecase.CreateClienteInput{
		Nome:            "Carlos Normal",
		Email:           "carlos@example.com",
		TipoSolicitacao: "Consulta",
		ValorPatrimonio: 199_999,
	})
	require.NoError(t, err)

	err = uc.ProcessWebhookCardUpdated(domainUsecase.WebhookCardUpdatedInput{
		EventID:      "evt_002",
		CardID:       "card_002",
		ClienteEmail: "carlos@example.com",
		Timestamp:    "2026-05-18T12:00:00Z",
	})

	require.NoError(t, err)
	assert.Equal(t, entity.StatusProcessado, repo.clientes[0].Status)
	assert.Equal(t, entity.PrioridadeNormal, repo.clientes[0].Prioridade)
}

// Teste 3 — event_id duplicado deve ser ignorado silenciosamente (idempotencia).
func TestProcessWebhook_EventIdDuplicado(t *testing.T) {
	repo := &mockClienteRepo{}
	uc := newUseCase(repo)

	_, err := uc.CreateCliente(domainUsecase.CreateClienteInput{
		Nome:            "Ana Idem",
		Email:           "ana@example.com",
		TipoSolicitacao: "Resgate",
		ValorPatrimonio: 300_000,
	})
	require.NoError(t, err)

	webhookInput := domainUsecase.WebhookCardUpdatedInput{
		EventID:      "evt_duplo_001",
		CardID:       "card_003",
		ClienteEmail: "ana@example.com",
		Timestamp:    "2026-05-18T12:00:00Z",
	}

	// Primeira execucao: deve processar normalmente.
	err = uc.ProcessWebhookCardUpdated(webhookInput)
	require.NoError(t, err)
	assert.Equal(t, entity.StatusProcessado, repo.clientes[0].Status)

	// Reverte o status para provar que a segunda chamada nao reprocessa.
	repo.clientes[0].Status = entity.StatusAguardandoAnalise

	// Segunda execucao com mesmo event_id: deve ser ignorada.
	err = uc.ProcessWebhookCardUpdated(webhookInput)
	require.NoError(t, err, "event_id duplicado nao deve retornar erro")

	// Status deve permanecer como revertido — nao foi reprocessado.
	assert.Equal(t, entity.StatusAguardandoAnalise, repo.clientes[0].Status)
}

// Teste 4 — Criar cliente com e-mail duplicado deve retornar apperror.Conflict (HTTP 409).
func TestCreateCliente_EmailDuplicado(t *testing.T) {
	repo := &mockClienteRepo{}
	uc := newUseCase(repo)

	input := domainUsecase.CreateClienteInput{
		Nome:            "Pedro Dup",
		Email:           "pedro@example.com",
		TipoSolicitacao: "Consulta",
		ValorPatrimonio: 50_000,
	}

	// Primeiro create: sucesso
	_, err := uc.CreateCliente(input)
	require.NoError(t, err)

	// Segundo create: repositório retorna Conflict (simula unique constraint do banco)
	repo.createErr = apperror.Conflict("Create cliente: registro duplicado (constraint: clientes_email_key)")

	_, err = uc.CreateCliente(input)
	require.Error(t, err)
	assert.True(t, apperror.IsConflict(err), "esperado apperror.Conflict para e-mail duplicado")
}

// Teste 5 — ProcessWebhook com e-mail inexistente deve retornar apperror.NotFound (HTTP 404).
func TestProcessWebhook_ClienteNaoEncontrado(t *testing.T) {
	repo := &mockClienteRepo{}
	uc := newUseCase(repo)

	err := uc.ProcessWebhookCardUpdated(domainUsecase.WebhookCardUpdatedInput{
		EventID:      "evt_404",
		CardID:       "card_404",
		ClienteEmail: "naoexiste@example.com",
		Timestamp:    "2026-05-18T12:00:00Z",
	})

	require.Error(t, err)
	assert.True(t, apperror.IsNotFound(err), "esperado apperror.NotFound para cliente inexistente")
}
