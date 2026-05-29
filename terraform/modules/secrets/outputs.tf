output "db_secret_arn" {
  description = "ARN do secret com credenciais do banco (usado pelo RDS Proxy e ECS)"
  value       = aws_secretsmanager_secret.db.arn
}

output "pipefy_secret_arn" {
  description = "ARN do secret com o token Pipefy"
  value       = aws_secretsmanager_secret.pipefy.arn
}
