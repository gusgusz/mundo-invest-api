# ─────────────────────────────────────────────────────────────────────────────
# Terraform Root — Mundo Invest API
#
# Orquestra os módulos em ordem de dependência:
#   secrets → vpc → ecr → rds → alb → ecs
#
# Pré-requisitos manuais (feitos UMA vez fora do Terraform):
#   1. Certificado TLS no ACM (var.acm_certificate_arn)
#   2. Bucket S3 + tabela DynamoDB para backend de state (opcional)
#   3. Credenciais AWS configuradas (aws configure ou OIDC no CI)
# ─────────────────────────────────────────────────────────────────────────────

# ── Secrets Manager (primeiro — outros módulos precisam dos ARNs) ───────────
module "secrets" {
  source = "./modules/secrets"

  project            = var.project
  db_username        = var.db_username
  db_password        = var.db_password
  db_name            = var.db_name
  db_proxy_endpoint  = "" # será atualizado após o RDS criar o proxy (apply em 2 passos ou use `depends_on`)
  pipefy_token       = var.pipefy_token
}

# ── Networking ───────────────────────────────────────────────────────────────
module "vpc" {
  source = "./modules/vpc"

  project            = var.project
  vpc_cidr           = var.vpc_cidr
  availability_zones = var.availability_zones
}

# ── ECR (registry de imagens) ────────────────────────────────────────────────
module "ecr" {
  source = "./modules/ecr"

  repository_name = "${var.project}-api"
}

# ── RDS PostgreSQL + Proxy ───────────────────────────────────────────────────
module "rds" {
  source = "./modules/rds"

  project            = var.project
  vpc_id             = module.vpc.vpc_id
  vpc_cidr           = var.vpc_cidr
  private_subnet_ids = module.vpc.private_subnet_ids

  db_name            = var.db_name
  db_username        = var.db_username
  db_password        = var.db_password
  db_secret_arn      = module.secrets.db_secret_arn

  db_instance_class    = var.db_instance_class
  allocated_storage    = var.db_allocated_storage
  multi_az             = var.environment == "production"
  deletion_protection  = var.environment == "production"

  depends_on = [module.vpc, module.secrets]
}

# ── Application Load Balancer ────────────────────────────────────────────────
module "alb" {
  source = "./modules/alb"

  project             = var.project
  vpc_id              = module.vpc.vpc_id
  public_subnet_ids   = module.vpc.public_subnet_ids
  container_port      = var.container_port
  acm_certificate_arn = var.acm_certificate_arn

  enable_deletion_protection = var.environment == "production"

  depends_on = [module.vpc]
}

# ── ECS Fargate ──────────────────────────────────────────────────────────────
module "ecs" {
  source = "./modules/ecs"

  project               = var.project
  aws_region            = var.aws_region
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  alb_security_group_id = module.alb.alb_security_group_id
  target_group_arn      = module.alb.target_group_arn

  ecr_repository_url    = module.ecr.repository_url
  image_tag             = var.image_tag

  container_port        = var.container_port
  task_cpu              = var.task_cpu
  task_memory           = var.task_memory
  desired_count         = var.desired_count
  min_capacity          = var.min_capacity
  max_capacity          = var.max_capacity

  db_secret_arn         = module.secrets.db_secret_arn
  pipefy_secret_arn     = module.secrets.pipefy_secret_arn
  pipefy_pipe_id        = var.pipefy_pipe_id

  depends_on = [module.alb, module.rds, module.secrets]
}
