# ─────────────────────────────────────────────────────────────────────────────
# Makefile — Mundo Invest API
# Atalhos para desenvolvimento, testes e infraestrutura
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: help dev test test-race build lint docker-build docker-up docker-down \
        tf-init tf-plan tf-apply tf-destroy tf-fmt tf-validate

GOTOOLCHAIN := GOTOOLCHAIN=local
BINARY      := server
AWS_REGION  := us-east-1

## help: Lista todos os comandos disponíveis
help:
	@grep -E '^##' Makefile | sed 's/## //'

# ── Desenvolvimento local ─────────────────────────────────────────────────────

## dev: Sobe o banco PostgreSQL e inicia a API com hot reload
dev: docker-up
	$(GOTOOLCHAIN) go run ./cmd/api

## test: Roda todos os testes unitários
test:
	$(GOTOOLCHAIN) go test ./tests/... -v

## test-race: Testes com detector de race conditions
test-race:
	$(GOTOOLCHAIN) go test ./tests/... -v -race

## test-coverage: Testes + relatório de cobertura no browser
test-coverage:
	$(GOTOOLCHAIN) go test ./tests/... -coverprofile=coverage.out
	$(GOTOOLCHAIN) go tool cover -html=coverage.out

## build: Compila o binário
build:
	$(GOTOOLCHAIN) go build -o $(BINARY) ./cmd/api

## lint: Roda o linter
lint:
	golangci-lint run ./...

# ── Docker ────────────────────────────────────────────────────────────────────

## docker-build: Constrói a imagem Docker de produção
docker-build:
	docker build -t mundo-invest-api:local .

## docker-up: Sobe o PostgreSQL local (docker compose)
docker-up:
	docker compose up -d

## docker-down: Para e remove os containers locais
docker-down:
	docker compose down

## docker-logs: Acompanha os logs da API
docker-logs:
	docker compose logs -f

# ── Terraform ─────────────────────────────────────────────────────────────────

## tf-init: Inicializa o Terraform no diretório terraform/
tf-init:
	cd terraform && terraform init

## tf-fmt: Formata todos os arquivos .tf
tf-fmt:
	cd terraform && terraform fmt -recursive

## tf-validate: Valida a configuração Terraform
tf-validate:
	cd terraform && terraform validate

## tf-plan: Mostra o plano de execução (sem aplicar)
tf-plan:
	cd terraform && terraform plan -out=tfplan

## tf-apply: Aplica as mudanças na AWS (pede confirmação)
tf-apply:
	cd terraform && terraform apply tfplan

## tf-destroy: DESTRÓI toda a infraestrutura (cuidado!)
tf-destroy:
	@echo "⚠️  ATENÇÃO: isso vai destruir TODA a infraestrutura na AWS."
	@read -p "Digite 'destruir' para confirmar: " confirm; \
	[ "$$confirm" = "destruir" ] && cd terraform && terraform destroy || echo "Cancelado."

# ── AWS (helpers pós-deploy) ─────────────────────────────────────────────────

## ecr-login: Autentica o docker no ECR
ecr-login:
	aws ecr get-login-password --region $(AWS_REGION) | \
	docker login --username AWS --password-stdin \
	  $$(aws sts get-caller-identity --query Account --output text).dkr.ecr.$(AWS_REGION).amazonaws.com

## ecs-logs: Acompanha os logs da task ECS em produção
ecs-logs:
	aws logs tail /ecs/mundo-invest --follow --region $(AWS_REGION)

## ecs-status: Status atual do serviço ECS
ecs-status:
	aws ecs describe-services \
	  --cluster mundo-invest-cluster \
	  --services mundo-invest-service \
	  --region $(AWS_REGION) \
	  --query 'services[0].{status:status,running:runningCount,desired:desiredCount,deployments:deployments[*].{id:id,status:status,running:runningCount}}'
