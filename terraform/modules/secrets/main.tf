# ─────────────────────────────────────────────────────────────────────────────
# Secrets Manager Module
# Armazena credenciais do banco e token do Pipefy de forma segura.
# A aplicação lê estes secrets via variáveis de ambiente injetadas pelo ECS.
# ─────────────────────────────────────────────────────────────────────────────

# ── Credenciais do banco (lidas pelo RDS Proxy e pela aplicação) ────────────
resource "aws_secretsmanager_secret" "db" {
  name                    = "${var.project}/db/credentials"
  description             = "Credenciais PostgreSQL (username + password)"
  recovery_window_in_days = 7

  tags = merge(var.tags, { Name = "${var.project}-secret-db" })
}

resource "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_secretsmanager_secret.db.id

  # Formato JSON compatível com o RDS Proxy
  secret_string = jsonencode({
    username = var.db_username
    password = var.db_password
    host     = var.db_proxy_endpoint
    port     = 5432
    dbname   = var.db_name
  })
}

# ── Token do Pipefy ────────────────────────────────────────────────────────
resource "aws_secretsmanager_secret" "pipefy" {
  name                    = "${var.project}/pipefy/token"
  description             = "Personal Access Token da API Pipefy"
  recovery_window_in_days = 7

  tags = merge(var.tags, { Name = "${var.project}-secret-pipefy" })
}

resource "aws_secretsmanager_secret_version" "pipefy" {
  secret_id     = aws_secretsmanager_secret.pipefy.id
  secret_string = jsonencode({ token = var.pipefy_token })
}
