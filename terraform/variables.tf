# ─────────────────────────────────────────────────────────────────────────────
# Variáveis do root — preencha no arquivo terraform.tfvars (nunca commite!)
# ─────────────────────────────────────────────────────────────────────────────

# ── Projeto / Ambiente ───────────────────────────────────────────────────────
variable "project" {
  description = "Prefixo usado em todos os recursos AWS"
  type        = string
  default     = "mundo-invest"
}

variable "environment" {
  description = "Ambiente de deploy (production | staging)"
  type        = string
  default     = "production"

  validation {
    condition     = contains(["production", "staging"], var.environment)
    error_message = "environment deve ser 'production' ou 'staging'."
  }
}

# ── AWS ──────────────────────────────────────────────────────────────────────
variable "aws_region" {
  description = "Região AWS onde subir a infraestrutura"
  type        = string
  default     = "us-east-1"
}

variable "availability_zones" {
  description = "AZs para subnets (mínimo 2 para Multi-AZ)"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}

# ── ACM ──────────────────────────────────────────────────────────────────────
variable "acm_certificate_arn" {
  description = "ARN do certificado TLS emitido pelo AWS Certificate Manager"
  type        = string
  # Obtenha com: aws acm list-certificates --region us-east-1
}

# ── Banco de dados ────────────────────────────────────────────────────────────
variable "db_name" {
  type    = string
  default = "mundoinvest"
}

variable "db_username" {
  type      = string
  sensitive = true
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "db_instance_class" {
  type    = string
  default = "db.t3.medium"
}

variable "db_allocated_storage" {
  type    = number
  default = 20
}

# ── Pipefy ───────────────────────────────────────────────────────────────────
variable "pipefy_token" {
  description = "Personal Access Token do Pipefy (Settings → Account → API)"
  type        = string
  sensitive   = true
}

variable "pipefy_pipe_id" {
  description = "ID do Pipe no Pipefy (último número da URL do Pipe)"
  type        = string
}

# ── ECS Fargate ──────────────────────────────────────────────────────────────
variable "container_port" {
  type    = number
  default = 8080
}

variable "image_tag" {
  description = "Tag da imagem Docker no ECR (atualizada pelo CI/CD)"
  type        = string
  default     = "latest"
}

variable "task_cpu" {
  description = "CPU units para a task (256 = 0.25 vCPU)"
  type        = string
  default     = "256"
}

variable "task_memory" {
  description = "Memória em MiB para a task"
  type        = string
  default     = "512"
}

variable "desired_count" {
  description = "Número inicial de tasks ECS"
  type        = number
  default     = 2
}

variable "min_capacity" {
  type    = number
  default = 1
}

variable "max_capacity" {
  type    = number
  default = 10
}
