output "repository_url" {
  description = "URL completa do repositório ECR (usada no docker push)"
  value       = aws_ecr_repository.app.repository_url
}

output "repository_arn" {
  value = aws_ecr_repository.app.arn
}
