# ─────────────────────────────────────────────────────────────────────────────
# ECR Module
# Elastic Container Registry para armazenar as imagens Docker da API
# ─────────────────────────────────────────────────────────────────────────────

resource "aws_ecr_repository" "app" {
  name                 = var.repository_name
  image_tag_mutability = "MUTABLE" # permite sobreescrever :latest no CI
  force_delete         = false     # produção: não apagar acidentalmente

  image_scanning_configuration {
    scan_on_push = true # detecta CVEs automaticamente a cada push
  }

  encryption_configuration {
    encryption_type = "AES256"
  }

  tags = merge(var.tags, { Name = var.repository_name })
}

# ── Lifecycle: mantém somente as últimas N imagens untagged ───────────────
resource "aws_ecr_lifecycle_policy" "app" {
  repository = aws_ecr_repository.app.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Remover imagens untagged após 7 dias"
        selection = {
          tagStatus   = "untagged"
          countType   = "sinceImagePushed"
          countUnit   = "days"
          countNumber = 7
        }
        action = { type = "expire" }
      },
      {
        rulePriority = 2
        description  = "Manter somente as últimas 20 imagens tagged"
        selection = {
          tagStatus     = "tagged"
          tagPrefixList = ["v", "sha-"]
          countType     = "imageCountMoreThan"
          countNumber   = 20
        }
        action = { type = "expire" }
      }
    ]
  })
}
