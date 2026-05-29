# ─────────────────────────────────────────────────────────────────────────────
# RDS Module
# PostgreSQL 16 em Multi-AZ, subnet group privada, RDS Proxy para pool de
# conexões (necessário com ECS Fargate que pode ter N tasks rodando)
# ─────────────────────────────────────────────────────────────────────────────

# ── Security Groups ─────────────────────────────────────────────────────────
resource "aws_security_group" "rds" {
  name        = "${var.project}-sg-rds"
  description = "Permite conexao PostgreSQL somente de dentro da VPC"
  vpc_id      = var.vpc_id

  ingress {
    description = "PostgreSQL from VPC"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, { Name = "${var.project}-sg-rds" })
}

# ── Subnet Group (subnets privadas) ────────────────────────────────────────
resource "aws_db_subnet_group" "main" {
  name       = "${var.project}-db-subnet-group"
  subnet_ids = var.private_subnet_ids

  tags = merge(var.tags, { Name = "${var.project}-db-subnet-group" })
}

# ── Parâmetros customizados ─────────────────────────────────────────────────
resource "aws_db_parameter_group" "main" {
  name   = "${var.project}-pg16"
  family = "postgres16"

  parameter {
    name  = "log_connections"
    value = "1"
  }

  parameter {
    name  = "log_disconnections"
    value = "1"
  }

  parameter {
    name  = "log_min_duration_statement"
    value = "500" # loga queries acima de 500ms
  }

  tags = var.tags
}

# ── Instância RDS ───────────────────────────────────────────────────────────
resource "aws_db_instance" "main" {
  identifier = "${var.project}-postgres"

  engine         = "postgres"
  engine_version = "16"
  instance_class = var.db_instance_class

  db_name  = var.db_name
  username = var.db_username
  password = var.db_password # gerenciado pelo Secrets Manager (veja secrets module)

  allocated_storage     = var.allocated_storage
  max_allocated_storage = var.max_allocated_storage
  storage_type          = "gp3"
  storage_encrypted     = true

  multi_az               = var.multi_az
  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  parameter_group_name   = aws_db_parameter_group.main.name

  backup_retention_period = 7          # 7 dias de point-in-time recovery
  backup_window           = "03:00-04:00"
  maintenance_window      = "Mon:04:00-Mon:05:00"

  deletion_protection       = var.deletion_protection
  skip_final_snapshot       = !var.deletion_protection
  final_snapshot_identifier = var.deletion_protection ? "${var.project}-final-snapshot" : null

  performance_insights_enabled = true

  tags = merge(var.tags, { Name = "${var.project}-postgres" })
}

# ── IAM Role para RDS Proxy ─────────────────────────────────────────────────
data "aws_iam_policy_document" "rds_proxy_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["rds.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "rds_proxy" {
  name               = "${var.project}-rds-proxy-role"
  assume_role_policy = data.aws_iam_policy_document.rds_proxy_assume.json
  tags               = var.tags
}

data "aws_iam_policy_document" "rds_proxy_secrets" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [var.db_secret_arn]
  }
}

resource "aws_iam_role_policy" "rds_proxy_secrets" {
  name   = "${var.project}-rds-proxy-secrets"
  role   = aws_iam_role.rds_proxy.id
  policy = data.aws_iam_policy_document.rds_proxy_secrets.json
}

# ── RDS Proxy ───────────────────────────────────────────────────────────────
resource "aws_db_proxy" "main" {
  name                   = "${var.project}-proxy"
  debug_logging          = false
  engine_family          = "POSTGRESQL"
  idle_client_timeout    = 1800
  require_tls            = true
  role_arn               = aws_iam_role.rds_proxy.arn
  vpc_security_group_ids = [aws_security_group.rds.id]
  vpc_subnet_ids         = var.private_subnet_ids

  auth {
    auth_scheme = "SECRETS"
    iam_auth    = "DISABLED"
    secret_arn  = var.db_secret_arn
  }

  tags = merge(var.tags, { Name = "${var.project}-proxy" })
}

resource "aws_db_proxy_default_target_group" "main" {
  db_proxy_name = aws_db_proxy.main.name

  connection_pool_config {
    max_connections_percent = 90
  }
}

resource "aws_db_proxy_target" "main" {
  db_instance_identifier = aws_db_instance.main.id
  db_proxy_name          = aws_db_proxy.main.name
  target_group_name      = aws_db_proxy_default_target_group.main.name
}
