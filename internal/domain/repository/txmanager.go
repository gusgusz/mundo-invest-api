package repository

import "context"

// TxManager define o contrato para execução de múltiplas operações de repositório
// em uma única transação de banco de dados atômica.
//
// Se fn retornar qualquer erro, toda a transação é revertida (rollback).
// Se fn retornar nil, a transação é confirmada (commit).
//
// O fn recebe um ClienteRepository escopado à transação — todas as operações
// feitas através dele participam da mesma transação.
type TxManager interface {
	RunInTx(ctx context.Context, fn func(ClienteRepository) error) error
}
