package model

import "time"

// ClienteGORM é o modelo de persistência da entidade Cliente para o GORM.
// Mantido separado da entidade de domínio para isolar a dependência de ORM
// da camada de regras de negócio.
type ClienteGORM struct {
	ID              uint    `gorm:"primarykey;autoIncrement"`
	Nome            string  `gorm:"column:nome;not null"`
	Email           string  `gorm:"column:email;uniqueIndex;not null"`
	TipoSolicitacao string  `gorm:"column:tipo_solicitacao;not null"`
	ValorPatrimonio float64 `gorm:"column:valor_patrimonio;not null"`
	Status          string  `gorm:"column:status;not null"`
	Prioridade      string  `gorm:"column:prioridade"`
	// EventID armazena o último event_id processado para garantir idempotência.
	EventID   string    `gorm:"column:event_id;index"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName define explicitamente o nome da tabela no banco.
func (ClienteGORM) TableName() string { return "clientes" }
