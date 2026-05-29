# Mundo Invest API

API de gerenciamento de clientes com integração simulada ao Pipefy via GraphQL.

## Stack

| Camada | Tecnologia |
|---|---|
| Linguagem | Go 1.22+ |
| Framework HTTP | Fiber v2 |
| ORM | GORM |
| Banco de dados | PostgreSQL (Docker) |
| Testes | Testify |

## Estrutura de Diretórios

```
.
├── cmd/api/                          → Ponto de entrada (main.go)
├── internal/
│   ├── domain/
│   │   ├── entity/                   → Entidades e regras de negócio puras
│   │   ├── repository/               → Interfaces de persistência (contratos)
│   │   └── usecase/                  → Interfaces dos casos de uso + DTOs
│   ├── usecase/                      → Implementação dos casos de uso
│   └── infra/
│       ├── database/
│       │   ├── model/                → Modelos GORM (separados do domínio)
│       │   └── repository/           → Implementação GORM dos repositórios
│       ├── http/
│       │   └── handlers/             → Handlers Fiber
│       └── pipefy/                   → Montagem das mutations GraphQL
└── tests/usecase/                    → Testes unitários com mocks
```

## Como executar localmente

### Pré-requisitos

- [Go 1.22+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) e Docker Compose

### 1. Clone e configure variáveis de ambiente

```bash
git clone <url-do-repositorio>
cd mundo-invest-api
cp .env.example .env
```

### 2. Suba o banco de dados

```bash
docker compose up -d
```

### 3. Execute a API

```bash
GOTOOLCHAIN=local go run ./cmd/api
```

A API estará disponível em `http://localhost:3000`.

---

## Executar os testes

Os testes são unitários com mocks em memória — **não requerem banco de dados**.

```bash
GOTOOLCHAIN=local go test ./tests/... -v
```

Saída esperada:

```
=== RUN   TestCreateCliente_Success
--- PASS: TestCreateCliente_Success (0.00s)
=== RUN   TestCreateCliente_EmailInvalido
--- PASS: TestCreateCliente_EmailInvalido (0.00s)
=== RUN   TestProcessWebhook_PrioridadeAlta
--- PASS: TestProcessWebhook_PrioridadeAlta (0.00s)
=== RUN   TestProcessWebhook_PrioridadeNormal
--- PASS: TestProcessWebhook_PrioridadeNormal (0.00s)
=== RUN   TestProcessWebhook_EventIdDuplicado
--- PASS: TestProcessWebhook_EventIdDuplicado (0.00s)
=== RUN   TestCreateCliente_EmailDuplicado
--- PASS: TestCreateCliente_EmailDuplicado (0.00s)
=== RUN   TestProcessWebhook_ClienteNaoEncontrado
--- PASS: TestProcessWebhook_ClienteNaoEncontrado (0.00s)
ok      github.com/gusgusz/mundo-invest-api/tests/usecase
```

---

## Exemplos de requisição (curl)

### POST /clientes — Criação de cliente

```bash
curl -X POST http://localhost:3000/clientes \
  -H "Content-Type: application/json" \
  -d '{
    "cliente_nome": "João Silva",
    "cliente_email": "joao.silva@example.com",
    "tipo_solicitacao": "Atualização cadastral",
    "valor_patrimonio": 250000
  }'
```

**Resposta (201 Created):**
```json
{
  "id": 1,
  "cliente_nome": "João Silva",
  "cliente_email": "joao.silva@example.com",
  "tipo_solicitacao": "Atualização cadastral",
  "valor_patrimonio": 250000,
  "status": "Aguardando Análise",
  "created_at": "2026-05-24T10:00:00Z",
  "updated_at": "2026-05-24T10:00:00Z"
}
```

A mutation `createCard` montada é logada no terminal:
```
[PIPEFY] createCard payload:
{ "query": "mutation createCard($input: CreateCardInput!) { ... }", "variables": { "input": { "pipe_id": "...", ... } } }
```

---

### POST /webhooks/pipefy/card-updated — Simulação de webhook

```bash
curl -X POST http://localhost:3000/webhooks/pipefy/card-updated \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt_123",
    "card_id": "card_456",
    "cliente_email": "joao.silva@example.com",
    "timestamp": "2026-05-18T12:00:00Z"
  }'
```

**Resposta (200 OK):**
```json
{ "message": "ok" }
```

**Idempotência:** reenviar o mesmo `event_id` retorna `200 OK` sem reprocessar.

A mutation `updateCardField` montada é logada no terminal:
```
[PIPEFY] updateCardField payload:
{ "query": "mutation updateCardFields(...) { ... }", "variables": { "inputStatus": { ... }, "inputPrioridade": { ... } } }
```

---

## Regra de Prioridade

| Patrimônio | Prioridade |
|---|---|
| ≥ R$ 200.000 | `prioridade_alta` |
| < R$ 200.000 | `prioridade_normal` |

---

## Visão de Produção (AWS) — Opcional

### Arquitetura proposta

```
Cliente HTTP
     │
     ▼
API Gateway (REST)
     │
     ├─► POST /clientes
     │        │
     │        ▼
     │   Lambda: CreateCliente
     │        │
     │        ├─► RDS (PostgreSQL Aurora Serverless) — persistência
     │        └─► Secrets Manager — credenciais
     │
     └─► POST /webhooks/pipefy/card-updated
              │
              ▼
         SQS (FIFO Queue)  ←── garante idempotência por MessageDeduplicationId
              │
              ▼
         Lambda: ProcessWebhook
              │
              ├─► RDS — leitura/atualização do cliente
              └─► CloudWatch Logs — auditoria do payload Pipefy
```

### Escalabilidade

- **API Gateway + Lambda**: escala automaticamente por requisição, sem gerenciar servidores.
- **SQS FIFO**: a fila com `MessageDeduplicationId = event_id` garante idempotência a nível de infraestrutura, eliminando processamentos duplicados mesmo com retentativas do Pipefy.
- **RDS Aurora Serverless v2**: escala automaticamente conforme a carga; o connection pooling via **RDS Proxy** evita estouro de conexões no padrão Lambda.
- **Secrets Manager**: rotação automática de credenciais do banco, sem segredos no código.
- **CloudWatch + X-Ray**: observabilidade completa do fluxo de webhook para rastrear cada `event_id` end-to-end.
