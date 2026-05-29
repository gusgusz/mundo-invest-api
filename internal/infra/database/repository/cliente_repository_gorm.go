package repository

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/gusgusz/mundo-invest-api/internal/domain/apperror"
	"github.com/gusgusz/mundo-invest-api/internal/domain/entity"
	domainRepo "github.com/gusgusz/mundo-invest-api/internal/domain/repository"
	"github.com/gusgusz/mundo-invest-api/internal/infra/database/model"
)

// translateError converte erros de infraestrutura (GORM/Postgres) em AppErrors
// com semântica de domínio. Centraliza toda a lógica de tradução de erros de BD.
//
// Mapeamentos:
//   - Postgres 23505 (unique_violation) → apperror.Conflict
//   - Qualquer outro erro                → apperror.Internal
func translateError(err error, ctx string) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return apperror.Conflict(
				fmt.Sprintf("%s: registro duplicado (constraint: %s)", ctx, pgErr.ConstraintName),
			)
		}
	}
	return apperror.Internal(ctx+": erro ao acessar banco de dados", err)
}

type clienteRepositoryGORM struct {
	db *gorm.DB
}

// Garantia em tempo de compilação de que *clienteRepositoryGORM satisfaz a interface.
var _ domainRepo.ClienteRepository = (*clienteRepositoryGORM)(nil)

// NewClienteRepository cria uma nova instância do repositório GORM.
func NewClienteRepository(db *gorm.DB) domainRepo.ClienteRepository {
	return &clienteRepositoryGORM{db: db}
}

// Create persiste um novo cliente no banco e preenche ID e timestamps na entidade.
func (r *clienteRepositoryGORM) Create(cliente *entity.Cliente) error {
	m := toModel(cliente)
	if err := r.db.Create(m).Error; err != nil {
		return translateError(err, "Create cliente")
	}
	// Propaga os valores gerados pelo banco de volta para a entidade.
	cliente.ID = m.ID
	cliente.CreatedAt = m.CreatedAt
	cliente.UpdatedAt = m.UpdatedAt
	return nil
}

// FindByEmail busca um cliente pelo e-mail; retorna nil, nil se não encontrado.
func (r *clienteRepositoryGORM) FindByEmail(email string) (*entity.Cliente, error) {
	var m model.ClienteGORM
	result := r.db.Where("email = ?", email).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, translateError(result.Error, "FindByEmail")
	}
	return toEntity(&m), nil
}

// FindByEventID verifica se o event_id já foi processado; retorna nil, nil se não encontrado.
func (r *clienteRepositoryGORM) FindByEventID(eventID string) (*entity.Cliente, error) {
	if eventID == "" {
		return nil, nil
	}
	var m model.ClienteGORM
	result := r.db.Where("event_id = ?", eventID).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, translateError(result.Error, "FindByEventID")
	}
	return toEntity(&m), nil
}

// Update persiste todas as alterações de um cliente existente.
func (r *clienteRepositoryGORM) Update(cliente *entity.Cliente) error {
	m := toModel(cliente)
	if err := r.db.Save(m).Error; err != nil {
		return translateError(err, "Update cliente")
	}
	return nil
}

// ── Mapeamento Domínio ↔ Persistência ─────────────────────────────────────────

func toModel(c *entity.Cliente) *model.ClienteGORM {
	return &model.ClienteGORM{
		ID:              c.ID,
		Nome:            c.Nome,
		Email:           c.Email,
		TipoSolicitacao: c.TipoSolicitacao,
		ValorPatrimonio: c.ValorPatrimonio,
		Status:          string(c.Status),
		Prioridade:      string(c.Prioridade),
		EventID:         c.EventID,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

func toEntity(m *model.ClienteGORM) *entity.Cliente {
	return &entity.Cliente{
		ID:              m.ID,
		Nome:            m.Nome,
		Email:           m.Email,
		TipoSolicitacao: m.TipoSolicitacao,
		ValorPatrimonio: m.ValorPatrimonio,
		Status:          entity.Status(m.Status),
		Prioridade:      entity.Prioridade(m.Prioridade),
		EventID:         m.EventID,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}
