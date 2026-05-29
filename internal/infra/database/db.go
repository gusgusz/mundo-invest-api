package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gusgusz/mundo-invest-api/internal/infra/database/model"
)

// Connect abre a conexão com o PostgreSQL e executa AutoMigrate da tabela de clientes.
// As credenciais são lidas das variáveis de ambiente (carregadas via .env pelo main).
func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Sao_Paulo",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao banco de dados: %w", err)
	}

	if err := db.AutoMigrate(&model.ClienteGORM{}); err != nil {
		return nil, fmt.Errorf("falha no AutoMigrate: %w", err)
	}

	return db, nil
}
