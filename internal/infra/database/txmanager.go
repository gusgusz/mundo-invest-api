package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	domainRepo "github.com/gusgusz/mundo-invest-api/internal/domain/repository"
	infraRepo "github.com/gusgusz/mundo-invest-api/internal/infra/database/repository"
)

// gormTxManager implementa domainRepo.TxManager usando GORM.
type gormTxManager struct {
	db *gorm.DB
}

// Garantia em tempo de compilação.
var _ domainRepo.TxManager = (*gormTxManager)(nil)

// NewTxManager cria um TxManager backed por um *gorm.DB.
func NewTxManager(db *gorm.DB) domainRepo.TxManager {
	return &gormTxManager{db: db}
}

// RunInTx abre uma transação, passa um repositório escopado a ela para fn,
// e faz commit em caso de sucesso ou rollback em caso de erro (inclusive panic).
//
// Garante que:
//   - Toda operação dentro de fn participa da mesma transação.
//   - Qualquer erro (retorno ou panic) aciona rollback automático.
//   - O commit só ocorre se fn retornar nil.
func (m *gormTxManager) RunInTx(ctx context.Context, fn func(domainRepo.ClienteRepository) error) error {
	tx := m.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("falha ao iniciar transação: %w", tx.Error)
	}

	// Rollback automático em caso de panic — evita transações abertas.
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-panic após o rollback
		}
	}()

	// Cria um repositório escopado à transação.
	txRepo := infraRepo.NewClienteRepository(tx)

	if err := fn(txRepo); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("falha ao commitar transação: %w", err)
	}

	return nil
}
