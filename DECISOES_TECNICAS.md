# Decisões Técnicas — Mundo Invest API

> Documento explicando **por que** cada tecnologia e padrão foi escolhido, **o que** cada parte do código faz, e a verificação completa dos requisitos do teste técnico.

---

## Índice

1. [Por que Go?](#1-por-que-go)
2. [Por que essa arquitetura?](#2-por-que-essa-arquitetura)
3. [O que é GraphQL e por que ele aparece aqui?](#3-o-que-é-graphql-e-por-que-ele-aparece-aqui)
4. [O que cada arquivo faz](#4-o-que-cada-arquivo-faz)
5. [Fluxo completo de uma requisição](#5-fluxo-completo-de-uma-requisição)
6. [Como localizar as mutations na documentação do Pipefy](#6-como-localizar-as-mutations-na-documentação-do-pipefy)
7. [Verificação dos requisitos do teste](#7-verificação-dos-requisitos-do-teste)
8. [TxManager — transações atômicas com rollback garantido](#8-txmanager--transações-atômicas-com-rollback-garantido)
9. [AppError — centralização e tipagem de erros](#9-apperror--centralização-e-tipagem-de-erros)
10. [Infraestrutura AWS com Terraform](#10-infraestrutura-aws-com-terraform)
11. [CI/CD com GitHub Actions](#11-cicd-com-github-actions)

---

## 1. Por que Go?

### O que é Go?

Go (também chamado de Golang) é uma linguagem criada pelo Google em 2009. É compilada (vira um binário executável), tem tipagem estática (os tipos são verificados antes de rodar) e foi feita especificamente para construir servidores e APIs de alta performance com código simples.

### Motivos da escolha

| Critério | Motivo |
|---|---|
| **Performance** | Go compila para binário nativo — é 5 a 10x mais rápido que Python em workloads de servidor |
| **Tipagem estática** | Erros como passar um `int` onde se espera `string` são pegos em tempo de compilação, não em produção |
| **Concorrência nativa** | Go tem `goroutines` — threads leves que permitem processar milhares de requisições simultâneas com poucos recursos |
| **Binário único** | O `go build` gera um único arquivo executável sem dependências externas — deploy simples |
| **Ecossistema maduro para APIs** | Fiber, GORM e Testify são bibliotecas de produção amplamente usadas no mercado |
| **Adequação ao teste** | O enunciado pede explicitamente Golang como opção, e é a stack da vaga |

### Como Go funciona na prática (resumo rápido)

```go
// Em Go, você define tipos explícitos:
type Status string                         // "Status" é uma string com nome
const StatusProcessado Status = "Processado"  // constante tipada

// Funções retornam múltiplos valores — o segundo geralmente é um erro:
func NewCliente(nome string) (*Cliente, error) {
    if nome == "" {
        return nil, errors.New("nome obrigatório")  // retorna nil + erro
    }
    return &Cliente{Nome: nome}, nil  // retorna o cliente + nil (sem erro)
}

// Quem chama DEVE tratar o erro:
cliente, err := NewCliente("")
if err != nil {
    // tratar o erro aqui
}
```

---

## 2. Por que essa arquitetura?

### O padrão escolhido: Clean Architecture + DDD

A estrutura do projeto segue dois padrões amplamente reconhecidos no mercado:

- **Clean Architecture** (Robert Martin / "Uncle Bob"): organiza o código em camadas concêntricas onde as regras de negócio ficam no centro e as dependências tecnológicas (banco, HTTP, Pipefy) ficam na borda.
- **DDD (Domain-Driven Design)**: a linguagem e a estrutura do código refletem o domínio do negócio — clientes, patrimônios, status de atendimento.

### A estrutura de diretórios explicada

```
mundo-invest-api/
│
├── cmd/api/main.go                    ← PONTO DE ENTRADA
│                                        Liga todas as peças e inicia o servidor
│
├── internal/
│   │
│   ├── domain/                        ← NÚCLEO — regras de negócio puras
│   │   ├── entity/cliente.go            Entidade Cliente com suas regras
│   │   ├── repository/                  CONTRATO: "preciso salvar/buscar clientes"
│   │   └── usecase/                     CONTRATO: "preciso criar cliente / processar webhook"
│   │
│   ├── usecase/                       ← ORQUESTRAÇÃO — coordena o fluxo
│   │   └── cliente_usecase_impl.go      Implementa os casos de uso usando os contratos
│   │
│   └── infra/                         ← BORDAS — detalhes técnicos substituíveis
│       ├── database/                    Conexão e repositório GORM (PostgreSQL)
│       ├── http/handlers/               Handlers Fiber (recebe JSON, devolve JSON)
│       └── pipefy/                      Monta as strings GraphQL do Pipefy
│
└── tests/usecase/                     ← TESTES — validam as regras sem banco real
```

### Por que separar assim?

**Problema**: se você mistura tudo (banco + regra de negócio + HTTP num arquivo só), qualquer mudança quebra tudo. Ex: trocar PostgreSQL por DynamoDB exigiria mexer nas regras de negócio.

**Solução com essa arquitetura**:

```
┌──────────────────────────────────────────────────┐
│  DOMÍNIO (entity, repository interface, usecase) │  ← Não sabe nada de banco, HTTP ou Pipefy
│  ↑ As camadas externas dependem DAQUI             │
└──────────────────────────────────────────────────┘
           ↑               ↑               ↑
    [GORM/Postgres]   [Fiber HTTP]   [Pipefy GraphQL]
    Pode ser trocado  Pode ser trocado  Pode ser real amanhã
```

**Na prática do teste**: os testes rodam 100% sem banco de dados, pois usam um "repositório falso" (`mockClienteRepo`) que implementa a mesma interface. Isso prova que a regra de negócio está isolada.

### Por que interfaces (contratos)?

Em Go, uma **interface** é uma lista de métodos que qualquer tipo pode implementar. É o mecanismo de isolamento:

```go
// Contrato: "qualquer coisa que tenha esses 4 métodos é um repositório de cliente"
type ClienteRepository interface {
    Create(cliente *entity.Cliente) error
    FindByEmail(email string) (*entity.Cliente, error)
    FindByEventID(eventID string) (*entity.Cliente, error)
    Update(cliente *entity.Cliente) error
}
```

O `UseCase` usa esse contrato. Em produção, passa o GORM. Nos testes, passa um mock em memória. O `UseCase` não sabe a diferença — e não precisa saber.

---

## 3. O que é GraphQL e por que ele aparece aqui?

### GraphQL em 60 segundos

GraphQL é uma linguagem de consulta para APIs criada pelo Facebook. Em vez de ter vários endpoints REST (`GET /cards`, `POST /cards`, `PATCH /cards/123`), você tem **um único endpoint** que recebe uma "query" descrevendo exatamente o que você quer fazer.

Há dois tipos de operação:
- **Query**: leitura de dados (equivalente ao GET)
- **Mutation**: escrita/alteração de dados (equivalente ao POST/PUT/DELETE)

O Pipefy usa GraphQL como API principal. Para criar um card no Pipefy "de verdade", você enviaria um JSON assim para `https://api.pipefy.com/graphql`:

```json
{
  "query": "mutation createCard($input: CreateCardInput!) { createCard(input: $input) { card { id title } } }",
  "variables": {
    "input": {
      "pipe_id": "123456",
      "title": "João Silva",
      "fields_attributes": [
        { "field_id": "cliente_email", "field_value": ["joao@email.com"] }
      ]
    }
  }
}
```

### O que o código faz com o GraphQL

O teste pede para **montar** essa string no formato correto, mas **não enviar de verdade** (pois não temos credenciais do Pipefy). O arquivo [internal/infra/pipefy/client.go](internal/infra/pipefy/client.go) faz exatamente isso: monta o payload e o imprime no terminal com `log.Printf`.

**Mutation 1 — `createCard`** (ao criar um cliente via `POST /clientes`):
```graphql
mutation createCard($input: CreateCardInput!) {
  createCard(input: $input) {
    card {
      id
      title
    }
  }
}
```

**Mutation 2 — `updateCardField`** (ao processar o webhook via `POST /webhooks/pipefy/card-updated`):
```graphql
mutation updateCardFields(
  $inputStatus:     UpdateCardFieldInput!,
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
}
```

> O uso de **aliases** (`updateStatus:`, `updatePrioridade:`) permite chamar a mesma mutation (`updateCardField`) duas vezes em uma única requisição — é uma funcionalidade nativa da especificação GraphQL, validada pela documentação do Pipefy.

---

## 4. O que cada arquivo faz

### `internal/domain/entity/cliente.go` — A entidade central

É o coração do sistema. Define **o que é um Cliente** e **quais são as regras sobre ele**.

| Responsabilidade | Como está no código |
|---|---|
| Estrutura de dados do cliente | `type Cliente struct { ... }` |
| Validar campos obrigatórios | `func (c *Cliente) Validate() error` |
| Validar formato de e-mail | `var emailRegex = regexp.MustCompile(...)` |
| Aplicar regra de prioridade | `func (c *Cliente) CalcularPrioridade()` |
| Executar transição de estado | `func (c *Cliente) Processar(eventID string)` |

A regra de prioridade vive **aqui**, não no banco nem no handler — porque ela é uma **regra de negócio**, não um detalhe técnico:

```go
func (c *Cliente) CalcularPrioridade() {
    if c.ValorPatrimonio >= 200_000 {
        c.Prioridade = PrioridadeAlta    // "prioridade_alta"
    } else {
        c.Prioridade = PrioridadeNormal  // "prioridade_normal"
    }
}
```

---

### `internal/domain/repository/cliente_repository.go` — O contrato do banco

Define **o que precisa ser possível fazer** com clientes no banco, sem dizer **como**:

```go
type ClienteRepository interface {
    Create(cliente *entity.Cliente) error
    FindByEmail(email string) (*entity.Cliente, error)
    FindByEventID(eventID string) (*entity.Cliente, error)  // ← garante idempotência
    Update(cliente *entity.Cliente) error
}
```

---

### `internal/domain/usecase/cliente_usecase.go` — Os contratos dos fluxos

Define os DTOs (objetos de transferência de dados) e as interfaces dos casos de uso:

- `CreateClienteInput` — o que chega no `POST /clientes`
- `WebhookCardUpdatedInput` — o que chega no `POST /webhooks/pipefy/card-updated`
- `CreateClienteOutput` — o que é retornado, incluindo a mutation GraphQL montada
- `ClienteUseCase` — interface com os dois métodos principais

---

### `internal/domain/usecase/pipefy_service.go` — O contrato do Pipefy

Interface que qualquer implementação do Pipefy deve satisfazer:

```go
type PipefyService interface {
    BuildCreateCardMutation(cliente *entity.Cliente) string
    BuildUpdateCardFieldMutation(cardID string, status entity.Status, prioridade entity.Prioridade) string
}
```

---

### `internal/usecase/cliente_usecase_impl.go` — A orquestração

Implementa os dois fluxos do teste. É aqui que "as coisas acontecem":

**Fluxo 1 — `CreateCliente`:**
```
Recebe input → Valida → Cria entidade → Salva no banco → Monta mutation Pipefy → Loga → Retorna
```

**Fluxo 2 — `ProcessWebhookCardUpdated`:**
```
Recebe input → Verifica event_id (idempotência) → Busca cliente → Aplica prioridade → Atualiza banco → Monta mutation Pipefy → Loga
```

---

### `internal/infra/pipefy/client.go` — As mutations GraphQL

Implementação concreta do `PipefyService`. Monta os payloads no formato exato exigido pela API do Pipefy (documentação: `developers.pipefy.com/reference/mutations-cards`).

**Não faz nenhuma requisição HTTP real** — apenas formata a string e retorna. Em produção, bastaria adicionar um `http.Post("https://api.pipefy.com/graphql", ...)` com o token Bearer.

---

### `internal/infra/database/model/cliente_model.go` — O modelo do banco

Separado da entidade de domínio de propósito. O GORM exige tags específicas (`gorm:"column:..."`) que poluiriam a entidade de domínio com detalhes de persistência. Esse arquivo faz a "tradução":

```
entity.Cliente  ←→  model.ClienteGORM  ←→  tabela "clientes" no PostgreSQL
```

---

### `internal/infra/database/repository/cliente_repository_gorm.go` — O repositório real

Implementa o contrato `ClienteRepository` usando GORM. Contém as funções `toModel()` e `toEntity()` que traduzem entre a entidade de domínio e o modelo do banco.

---

### `internal/infra/http/handlers/` — Os handlers HTTP

Recebem a requisição HTTP, fazem o binding do JSON para os DTOs e chamam o UseCase:

```
Request JSON → BodyParser → CreateClienteInput → UseCase.CreateCliente() → Response JSON
```

Não têm nenhuma regra de negócio — apenas validação de parsing e tradução HTTP↔UseCase.

---

### `cmd/api/main.go` — O ponto de entrada (Composition Root)

É aqui que todas as peças são montadas juntas:

```go
repo         := dbRepo.NewClienteRepository(db)           // repositório GORM real
pipefyClient := pipefy.NewClient(os.Getenv("PIPEFY_PIPE_ID"))  // cliente Pipefy real
uc           := usecase.NewClienteUseCase(repo, pipefyClient)   // use case com dependências reais
server.SetupRoutes(app, uc)  // registra rotas no Fiber
```

Este padrão se chama **Injeção de Dependência** manual — as dependências são construídas fora e passadas para dentro. Isso é o que permite que os testes passem mocks sem mudar nada no UseCase.

---

### `tests/usecase/cliente_usecase_test.go` — Os testes

5 testes unitários, todos sem banco de dados:

| Teste | O que valida |
|---|---|
| `TestCreateCliente_Success` | Cliente criado, status inicial correto, mutation retornada |
| `TestCreateCliente_EmailInvalido` | E-mail inválido gera erro, nada é salvo |
| `TestProcessWebhook_PrioridadeAlta` | Patrimônio `>= 200k` → `prioridade_alta` |
| `TestProcessWebhook_PrioridadeNormal` | Patrimônio `< 200k` → `prioridade_normal` |
| `TestProcessWebhook_EventIdDuplicado` | Mesmo `event_id` enviado duas vezes → segundo é ignorado |

---

## 5. Fluxo completo de uma requisição

### Exemplo: `POST /clientes`

```
1. Cliente HTTP envia JSON:
   { "cliente_nome": "João", "cliente_email": "joao@email.com", ... }

2. Fiber (framework HTTP) recebe a requisição
   └─► handlers/cliente_handler.go
       └─► c.BodyParser(&input) — converte JSON para struct Go

3. Handler chama o UseCase:
   └─► usecase/cliente_usecase_impl.go → CreateCliente(input)
       │
       ├─► 1. entity.NewCliente(...) — valida campos e e-mail
       │       Se inválido → retorna erro → handler responde 422
       │
       ├─► 2. repo.Create(cliente) — salva no PostgreSQL via GORM
       │       Se erro de banco → retorna erro → handler responde 500
       │
       └─► 3. pipefy.BuildCreateCardMutation(cliente) — monta GraphQL
               log.Printf("[PIPEFY] ...") — imprime no terminal
               Retorna o cliente salvo

4. Handler responde HTTP 201 com o JSON do cliente
```

---

## 6. Como localizar as mutations na documentação do Pipefy

### Onde fica a documentação

O Pipefy tem um portal público de desenvolvedores em `developers.pipefy.com`. A seção relevante é:

```
developers.pipefy.com
└── API Reference
    └── Mutations
        └── Cards   ← aqui estão createCard e updateCardField
```

### Passo a passo de como encontrar `createCard`

1. Acesse `developers.pipefy.com/reference/mutations-cards`
2. Na lista de mutations, procure por **`createCard`**
3. A página mostra:
   - O **tipo de input**: `CreateCardInput!` (o `!` significa obrigatório)
   - Os **campos aceitos**: `pipe_id`, `title`, `fields_attributes`
   - O **retorno**: `CreateCardPayload` com `card { id title }`
   - Um **exemplo de código** pronto

O que você vê na documentação é exatamente o que está no código:

```graphql
# Documentação do Pipefy mostra:
mutation {
  createCard(input: {
    pipe_id: "123"
    title: "Nome do card"
    fields_attributes: [
      { field_id: "campo_1", field_value: ["valor"] }
    ]
  }) {
    card {
      id
      title
    }
  }
}
```

A versão com variáveis (que usamos, pois é a forma parametrizável e segura) fica:

```graphql
mutation createCard($input: CreateCardInput!) {
  createCard(input: $input) {
    card { id  title }
  }
}
```

Com as variáveis separadas no JSON:
```json
{ "input": { "pipe_id": "...", "title": "...", "fields_attributes": [...] } }
```

### Passo a passo de como encontrar `updateCardField`

1. Na mesma página `developers.pipefy.com/reference/mutations-cards`
2. Procure por **`updateCardField`**
3. A documentação mostra o tipo `UpdateCardFieldInput` com os campos:
   - `card_id` — ID do card a atualizar
   - `field_id` — slug do campo
   - `new_value` — novo valor (string)

```graphql
# Exemplo da documentação:
mutation {
  updateCardField(input: {
    card_id: "456"
    field_id: "status_cliente"
    new_value: "Processado"
  }) {
    card { id }
    success
  }
}
```

### Por que usamos aliases para chamar `updateCardField` duas vezes?

O desafio era: precisamos atualizar **dois campos** (status + prioridade) em uma única requisição. GraphQL não permite chamar a mesma mutation duas vezes com o mesmo nome. A solução — documentada na especificação oficial do GraphQL — são os **aliases**:

```graphql
mutation updateCardFields(...) {
  updateStatus: updateCardField(input: $inputStatus) {      ← alias "updateStatus"
    card { id }
    success
  }
  updatePrioridade: updateCardField(input: $inputPrioridade) {  ← alias "updatePrioridade"
    card { id }
    success
  }
}
```

O servidor GraphQL trata cada alias como uma operação independente. Isso evita 2 requisições HTTP separadas.

### Como descobrir os `field_id` de um pipe real

Em um cenário real, você precisaria saber o `field_id` (slug) de cada campo customizado do pipe. Há duas formas:

**Opção 1 — Via documentação/interface do Pipefy:**
- No Pipefy, vá em configurações do pipe → campos → cada campo tem um "identificador" (slug), que é o `field_id`

**Opção 2 — Via query GraphQL:**
```graphql
query {
  pipe(id: "SEU_PIPE_ID") {
    start_form_fields {
      id       ← este é o field_id
      label
      type
    }
  }
}
```

Enviando essa query para `https://api.pipefy.com/graphql` com o header `Authorization: Bearer SEU_TOKEN`, o Pipefy retorna todos os campos e seus IDs.

No nosso projeto, os `field_id` (`cliente_email`, `tipo_solicitacao`, `valor_patrimonio`, `status_cliente`, `prioridade`) são **placeholders** que em produção seriam substituídos pelos slugs reais do pipe configurado no Pipefy do Mundo Invest.

---

## 7. Verificação dos requisitos do teste

### Requisitos funcionais

| Requisito | Arquivo | Status |
|---|---|---|
| `POST /clientes` exposto | [internal/infra/http/router.go](internal/infra/http/router.go) | ✅ |
| Validação de campos obrigatórios | [internal/domain/entity/cliente.go](internal/domain/entity/cliente.go) → `Validate()` | ✅ |
| Validação de e-mail | [internal/domain/entity/cliente.go](internal/domain/entity/cliente.go) → `emailRegex` | ✅ |
| Status inicial `"Aguardando Análise"` | [internal/domain/entity/cliente.go](internal/domain/entity/cliente.go) → `NewCliente()` | ✅ |
| Persistência no PostgreSQL | [internal/infra/database/repository/cliente_repository_gorm.go](internal/infra/database/repository/cliente_repository_gorm.go) | ✅ |
| Mutation `createCard` GraphQL (spec Pipefy) | [internal/infra/pipefy/client.go](internal/infra/pipefy/client.go) | ✅ |
| `POST /webhooks/pipefy/card-updated` exposto | [internal/infra/http/router.go](internal/infra/http/router.go) | ✅ |
| Idempotência por `event_id` | [internal/usecase/cliente_usecase_impl.go](internal/usecase/cliente_usecase_impl.go) → `FindByEventID` | ✅ |
| Regra: `>= 200k` → `prioridade_alta` | [internal/domain/entity/cliente.go](internal/domain/entity/cliente.go) → `CalcularPrioridade()` | ✅ |
| Regra: `< 200k` → `prioridade_normal` | [internal/domain/entity/cliente.go](internal/domain/entity/cliente.go) → `CalcularPrioridade()` | ✅ |
| Mutation `updateCardField` GraphQL (spec Pipefy) | [internal/infra/pipefy/client.go](internal/infra/pipefy/client.go) | ✅ |
| Status muda para `"Processado"` | [internal/domain/entity/cliente.go](internal/domain/entity/cliente.go) → `Processar()` | ✅ |
| Persistência da prioridade e status | [internal/infra/database/repository/cliente_repository_gorm.go](internal/infra/database/repository/cliente_repository_gorm.go) → `Update()` | ✅ |

### Testes obrigatórios

| Teste do enunciado | Teste no código | Status |
|---|---|---|
| Criação de cliente com payload válido | `TestCreateCliente_Success` | ✅ |
| Regra de prioridade correta por patrimônio | `TestProcessWebhook_PrioridadeAlta` + `TestProcessWebhook_PrioridadeNormal` | ✅ |
| Bloqueio de `event_id` duplicado | `TestProcessWebhook_EventIdDuplicado` | ✅ |

### README (Seção 4 do enunciado)

| Item exigido | Status |
|---|---|
| Instruções de execução local | ✅ [README.md](README.md) |
| Exemplos de `curl` para os dois endpoints | ✅ [README.md](README.md) |
| Visão de produção AWS (Lambda, API Gateway, RDS/DynamoDB) | ✅ [README.md](README.md) |

---

### Stack utilizada vs. enunciado

| Enunciado | Implementado |
|---|---|
| Python ou **Golang** | ✅ Go 1.22 |
| **Fiber** | ✅ `github.com/gofiber/fiber/v2` |
| **GORM** | ✅ `gorm.io/gorm` + `gorm.io/driver/postgres` |
| **Testify** | ✅ `github.com/stretchr/testify` |
| **PostgreSQL via Docker** | ✅ `docker-compose.yml` com `postgres:16-alpine` |
| Estrutura `/cmd/api`, `/internal/domain/entity`, `/internal/usecase`, `/internal/infra/*` | ✅ Idêntica ao especificado |

---

## 8. TxManager — transações atômicas com rollback garantido

### O problema que ele resolve

O fluxo do webhook executa duas consultas seguidas:

```
1. FindByEventID(event_id)  ← lê se já foi processado
2. Update(cliente)          ← grava prioridade + status
```

Sem transação, duas requisições HTTP concorrentes com o **mesmo `event_id`** podem:

```
Goroutine A: FindByEventID("evt_001") → nil (não encontrado ainda)
Goroutine B: FindByEventID("evt_001") → nil (não encontrado ainda)
Goroutine A: Update(cliente) ← processa
Goroutine B: Update(cliente) ← processa de novo — DUPLICATA!
```

Isso é uma **race condition TOCTOU** (Time-Of-Check-Time-Of-Use). A única forma de eliminá-la é fazer o check e o update dentro da **mesma transação com isolamento de leitura**, impedindo que outra transação enxergue o estado intermediário.

### Arquitetura do TxManager

```
internal/domain/repository/txmanager.go   ← interface (domínio puro)
internal/infra/database/txmanager.go      ← implementação GORM
```

A interface vive no domínio — o caso de uso depende dela:

```go
type TxManager interface {
    RunInTx(ctx context.Context, fn func(ClienteRepository) error) error
}
```

A implementação GORM fica na infra e é injetada pelo `main.go` (Composition Root). O domínio **nunca importa GORM**.

### O que `RunInTx` garante

```
fn retorna nil   → COMMIT   (todas as escritas são confirmadas)
fn retorna error → ROLLBACK (nenhuma escrita chega ao banco)
fn entra em panic → ROLLBACK via defer + re-panic (sem transação "presa" aberta)
```

```go
func (m *gormTxManager) RunInTx(ctx context.Context, fn func(ClienteRepository) error) error {
    tx := m.db.WithContext(ctx).Begin()

    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()  // sempre fecha a transação em caso de panic
            panic(r)
        }
    }()

    txRepo := NewClienteRepository(tx)  // repositório escopado à transação

    if err := fn(txRepo); err != nil {
        tx.Rollback()
        return err
    }
    return tx.Commit().Error
}
```

### Por que a interface fica no domínio e não na infra?

**Regra de dependência da Clean Architecture:** as camadas externas (infra) dependem das internas (domínio), nunca o contrário.

Se `TxManager` ficasse no pacote `infra/database`, o caso de uso teria que importar a infra para usar a interface — invertendo a hierarquia e acoplando o domínio à implementação concreta. Com a interface no domínio, o caso de uso depende apenas de uma abstração; a infra fornece a implementação.

### Nos testes

Como os testes são unitários (sem banco real), o `mockTxManager` simplesmente chama `fn(repo)` direto — sem transação real. O comportamento de negócio é idêntico; a diferença é só na camada de isolamento do banco, que os testes unitários não precisam verificar (isso fica nos testes de integração).

```go
type mockTxManager struct{ repo ClienteRepository }

func (m *mockTxManager) RunInTx(_ context.Context, fn func(ClienteRepository) error) error {
    return fn(m.repo)
}
```

---

## 9. AppError — centralização e tipagem de erros

### O problema que ele resolve

Antes desta mudança, os erros eram retornados como `error` genérico ou `fmt.Errorf(...)`. Isso criava dois problemas:

1. **O handler não sabia que HTTP status usar.** Qualquer erro do repositório virava HTTP 500 — mesmo um e-mail duplicado (que deveria ser 409) ou cliente não encontrado (que deveria ser 404).

2. **Mensagens de banco de dados vazavam para o cliente.** `"ERROR: duplicate key value violates unique constraint \"clientes_email_key\""` é uma mensagem interna do Postgres que o usuário final não deve ver.

### Estrutura do `AppError`

```go
type AppError struct {
    Code    ErrorCode  // "NOT_FOUND" | "CONFLICT" | "VALIDATION" | "INTERNAL"
    Message string     // mensagem segura para exibir ao cliente
    Cause   error      // erro original (para log interno), nunca exposto na API
}
```

O campo `Cause` implementa `Unwrap()`, o que significa que `errors.As` e `errors.Is` conseguem navegar pela cadeia de erros — compatível com o padrão idiomático do Go desde a versão 1.13.

### Fluxo do erro desde o banco até o HTTP

```
PostgreSQL
  └─ pgconn.PgError{Code: "23505", ConstraintName: "clientes_email_key"}
       │
       ▼ translateError() — no repositório GORM
  apperror.Conflict("Create cliente: registro duplicado (constraint: clientes_email_key)")
       │
       ▼ caso de uso propaga o erro sem modificar
  handler.CreateCliente()
       │
       ▼ errorResponse() — helper compartilhado entre handlers
  HTTP 409 { "error": "Create cliente: registro duplicado...", "code": "CONFLICT" }
```

### `translateError` — por que fica no repositório?

Somente o repositório conhece os detalhes de Postgres (`pgconn.PgError`, código `23505`). Se a tradução ficasse no caso de uso ou no handler, eles precisariam importar `pgconn`, acoplando a lógica de negócio ao driver de banco de dados — violando a Clean Architecture.

A tradução acontece na fronteira domínio ↔ infra: o repositório converte o erro técnico em semântica de domínio antes de devolvê-lo.

### `errorResponse` — centralização no handler

Antes, cada handler tinha seu próprio código de mapeamento:

```go
// antes — espalhado em cliente_handler.go e webhook_handler.go
return c.Status(fiber.StatusUnprocessableEntity).JSON(...)
return c.Status(fiber.StatusInternalServerError).JSON(...)
```

Agora, um único helper em `handlers/error.go` decide o status baseado no `Code`:

```go
func errorResponse(c *fiber.Ctx, err error) error {
    var appErr *apperror.AppError
    if errors.As(err, &appErr) {
        switch appErr.Code {
        case apperror.CodeNotFound:   return c.Status(404).JSON(...)
        case apperror.CodeConflict:   return c.Status(409).JSON(...)
        case apperror.CodeValidation: return c.Status(422).JSON(...)
        }
    }
    // Fallback: nunca expõe detalhes internos ao cliente
    return c.Status(500).JSON(fiber.Map{"error": "erro interno do servidor"})
}
```

**Benefício:** adicionar um novo `ErrorCode` no futuro exige alterar apenas `apperror/errors.go` e `handlers/error.go`. Todos os handlers se beneficiam automaticamente.

### Helpers de verificação

```go
apperror.IsConflict(err)    // true se err contém um AppError com Code == "CONFLICT"
apperror.IsNotFound(err)    // true se err contém um AppError com Code == "NOT_FOUND"
apperror.IsValidation(err)  // true se err contém um AppError com Code == "VALIDATION"
```

Usados nos testes para verificar o tipo do erro sem depender da mensagem de texto (que pode mudar):

```go
assert.True(t, apperror.IsConflict(err), "esperado Conflict para e-mail duplicado")
assert.True(t, apperror.IsNotFound(err), "esperado NotFound para cliente inexistente")
```

### Tabela de mapeamento completo

| Situação | `ErrorCode` | HTTP Status |
|---|---|---|
| E-mail ou campo inválido | `VALIDATION` | 422 Unprocessable Entity |
| E-mail já cadastrado (unique constraint) | `CONFLICT` | 409 Conflict |
| Cliente não encontrado | `NOT_FOUND` | 404 Not Found |
| Falha interna de banco ou infra | `INTERNAL` | 500 Internal Server Error |

---

## 10. Infraestrutura AWS com Terraform

### Por que Terraform?

Terraform é o padrão de fato para **Infrastructure as Code (IaC)** na AWS. A alternativa da AWS é o CDK (Cloud Development Kit), que usa TypeScript/Python. Escolhemos Terraform por:

| Critério | Terraform | AWS CDK |
|---|---|---|
| Linguagem | HCL declarativo (legível) | TypeScript/Python (imperativo) |
| Multi-cloud | Sim — mesma sintaxe para GCP, Azure | Não — amarrado à AWS |
| Estado da arte | Maduro, >10 anos, enorme comunidade | Mais novo, breaking changes frequentes |
| Módulos da comunidade | Registry.terraform.io (imensurável) | Constructs Library (menor) |
| Plano de mudança | `terraform plan` mostra o diff antes de aplicar | Não tem equivalente tão claro |

### Arquitetura escolhida: ECS Fargate (não Lambda)

A API usa **Fiber**, que é um servidor HTTP persistente (como Express no Node.js). Lambda é projetada para execução stateless e efêmera — ela não mantém um servidor ouvindo portas. Usar Lambda com Fiber exigiria um adapter extra (`aws-lambda-go-api-proxy`) com overhead e limitações.

**ECS Fargate** é a escolha correta porque:
- Container Docker roda igual ao ambiente de desenvolvimento (docker-compose → ECS, sem surpresas)
- Servidor HTTP persistente, sem cold start perceptível
- Auto Scaling horizontal nativo (1 a N tasks conforme carga)
- Suporta conexões de longa duração (webhooks, streaming)

### Diagrama da Infraestrutura

```
Internet
    │
    ▼
[ ALB ] — HTTPS only (TLS 1.3, HTTP→HTTPS redirect)
    │
    ├─ Target Group (Health Check: GET /health)
    │
    ▼
[ ECS Fargate Cluster ] — subnet privada (sem IP público)
    │   ┌──────────────┐   ┌──────────────┐
    │   │  Task (app)  │   │  Task (app)  │  ← Auto Scaling: FARGATE + FARGATE_SPOT
    │   └──────┬───────┘   └──────┬───────┘
    │          │                   │
    │          └─────────┬─────────┘
    │                    ▼
    │           [ RDS Proxy ]  ← pool de conexões (evita esgotar max_connections)
    │                    │
    │                    ▼
    │           [ RDS PostgreSQL 16 ]  ← Multi-AZ em produção
    │
    └─ Secrets Manager ← credenciais injetadas no container (nunca em env var plaintext)

[ ECR ] ← imagens Docker versionadas por SHA do commit
[ CloudWatch Logs ] ← /ecs/mundo-invest (retenção 30 dias)
[ NAT Gateway ] ← tasks privadas saem para internet (chamar API do Pipefy)
```

### Por que cada serviço AWS?

#### ECR (Elastic Container Registry)
- Registry privado gerenciado pela AWS, integrado nativamente ao ECS
- Scan de vulnerabilidades automático a cada `docker push`
- Lifecycle policy limpa imagens antigas automaticamente
- Alternativa (Docker Hub) exige gerenciar credenciais e tem rate limiting

#### RDS Proxy
O ECS pode escalar para N tasks simultâneas. Cada task abre conexões com o PostgreSQL. O RDS tem um limite de `max_connections` (~100 no `db.t3.medium`). Sem o proxy, 20 tasks × 5 conexões = 100 conexões → banco travado.

O **RDS Proxy** mantém um pool de conexões permanente com o banco e distribui entre as tasks, absorvendo picos de escala sem estressar o PostgreSQL.

#### Secrets Manager (não SSM Parameter Store nem variáveis de ambiente)
Hierarquia de segurança para segredos:

| Abordagem | Segurança | Rotação automática |
|---|---|---|
| `ENV` no Dockerfile | ❌ Vaza no histórico git | ❌ |
| SSM Parameter Store | ✅ Criptografado | Manual |
| **Secrets Manager** | ✅ Criptografado | ✅ Automático |

O ECS injeta os segredos como variáveis de ambiente **em runtime**, lendo diretamente do Secrets Manager. O código Go nunca vê a senha — apenas lê `os.Getenv("DB_PASSWORD")` que já foi populado.

#### FARGATE_SPOT
Tasks adicionais (além da mínima de base) rodam em capacidade SPOT, que é **até 70% mais barata** que FARGATE on-demand. O circuit breaker do ECS garante que, se uma task SPOT for interrompida pela AWS, uma nova sobe automaticamente.

#### NAT Gateway (único, não por AZ)
Em produção real, cada AZ teria seu próprio NAT Gateway (alta disponibilidade). Para o MVP/teste, usamos apenas um (na primeira AZ) para economizar custo (~$32/mês por NAT Gateway). O comentário no código documenta essa decisão.

### Módulos Terraform

```
terraform/
├── providers.tf              # AWS provider + backend S3 (comentado para ativar)
├── main.tf                   # Orquestra os módulos
├── variables.tf              # Todas as variáveis do root
├── outputs.tf                # ALB DNS, ECR URL, etc.
├── terraform.tfvars.example  # Template (commitar) — terraform.tfvars no .gitignore
└── modules/
    ├── vpc/      # VPC, subnets pub/priv, IGW, NAT, Route Tables
    ├── ecr/      # Registry de imagens + lifecycle policy
    ├── rds/      # PostgreSQL 16 + RDS Proxy + Security Group
    ├── alb/      # ALB + Target Group + Listener HTTPS + redirect HTTP
    ├── ecs/      # Cluster + Task Definition + Service + Auto Scaling + IAM
    └── secrets/  # Secrets Manager: credenciais DB + token Pipefy
```

### Como aplicar pela primeira vez

```bash
# 1. Preencher variáveis
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
# edite terraform.tfvars com suas credenciais reais

# 2. Inicializar
make tf-init

# 3. Validar
make tf-validate

# 4. Ver o plano (sem modificar nada)
make tf-plan

# 5. Aplicar
make tf-apply
```

> **Pré-requisito manual:** Emitir um certificado no AWS Certificate Manager (ACM) para o seu domínio e colocar o ARN em `acm_certificate_arn`. Isso não pode ser automatizado pelo Terraform porque a validação do domínio requer uma ação humana (clicar no e-mail ou criar um registro DNS).

---

## 11. CI/CD com GitHub Actions

### Por que GitHub Actions e não Jenkins/GitLab CI/CircleCI?

| Critério | GitHub Actions | Jenkins | CircleCI |
|---|---|---|---|
| Infraestrutura | Zero — gerenciado pelo GitHub | Precisa de servidor próprio | Gerenciado |
| Integração com repo | Nativa (mesmo lugar do código) | Webhook externo | Webhook externo |
| OIDC com AWS | Suportado nativamente | Plugin extra | Suportado |
| Custo | 2000 min/mês grátis | Custo de servidor | 6000 créditos grátis |
| Secrets | GitHub Secrets (criptografado) | Jenkins Credentials | CircleCI Contexts |

Para um projeto que vive no GitHub, Actions é a escolha mais simples e sem fricção.

### Dois workflows separados (CI ≠ CD)

A separação é intencional:

```
PRs e pushes em qualquer branch → CI (ci.yml)
                                    ├── Unit tests (com -race)
                                    ├── Lint (golangci-lint)
                                    ├── Build check
                                    └── Terraform validate

Apenas quando CI passa na main → CD (deploy.yml)
                                    ├── Build Docker image
                                    ├── Push para ECR (tag: sha-<commit>)
                                    ├── Render nova Task Definition
                                    └── Deploy ECS (aguarda estabilidade)
```

Isso garante que **código quebrado nunca chega em produção** — o deploy só roda depois que todos os testes e lint passaram.

### OIDC — sem chaves de acesso estáticas

A prática antiga era criar um IAM User, gerar `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` e colocar nos GitHub Secrets. Isso é um risco de segurança: essas chaves têm longa vida e, se vazarem, o atacante tem acesso à AWS.

**OIDC (OpenID Connect)** é a abordagem moderna:
1. GitHub gera um token JWT temporário para cada run do workflow
2. A AWS recebe esse token e verifica que vem do GitHub (repositório correto + branch correto)
3. O token é trocado por credenciais temporárias da AWS (válidas por 15 minutos)
4. Nenhuma chave de longa vida é armazenada em lugar nenhum

```yaml
permissions:
  id-token: write  # permite que o workflow solicite um token OIDC
  contents: read

- uses: aws-actions/configure-aws-credentials@v4
  with:
    role-to-assume: ${{ secrets.AWS_DEPLOY_ROLE_ARN }}
    aws-region: us-east-1
```

Para configurar: crie um IAM Identity Provider do tipo OpenID Connect no console AWS, com issuer `token.actions.githubusercontent.com`, e crie uma IAM Role com a política de trust que limita o acesso ao seu repositório específico.

### Rastreabilidade com SHA do commit

```
ECR: mundo-invest-api:sha-a1b2c3d4e5f6
               ↑
               └─ sha curto do commit que gerou a imagem
```

Isso permite:
- Rollback imediato: basta atualizar a Task Definition para apontar para um SHA anterior
- Auditoria completa: dado qualquer tag de imagem, sabe-se exatamente qual código está rodando
- A tag `:latest` é atualizada junto, mas nunca usada em produção diretamente

### Rollback automático via Circuit Breaker

No módulo ECS, o serviço tem:

```hcl
deployment_circuit_breaker {
  enable   = true
  rollback = true
}
```

Se as novas tasks não passarem no health check dentro do timeout, o ECS automaticamente:
1. Para o deploy
2. Reverte para a Task Definition anterior
3. O job `deploy.yml` falha com erro visível

O health check do container (`GET /health → 200`) é a porta de entrada desse mecanismo.

### Variáveis e fluxo completo

```
Desenvolvedor faz 'git push' na branch feature
        │
        ▼
[GitHub Actions: ci.yml]
  ├── go test ./tests/... -race     → falhou? PR bloqueado
  ├── golangci-lint                 → falhou? PR bloqueado
  ├── go build ./cmd/api            → falhou? PR bloqueado
  └── terraform validate            → falhou? PR bloqueado
        │
        ▼ (após merge na main)
[GitHub Actions: deploy.yml]
  ├── Autenticar na AWS via OIDC (token temporário, 15min)
  ├── docker build + push ECR (tag: sha-<commit>)
  ├── Render nova Task Definition com nova imagem
  ├── aws ecs deploy (rolling, zero downtime)
  └── Aguarda estabilidade — falhou? rollback automático
        │
        ▼
[ALB] serve a nova versão sem downtime
```
