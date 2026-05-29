output "alb_dns_name" {
  description = "DNS do Load Balancer — aponte seu registro CNAME aqui"
  value       = module.alb.alb_dns_name
}

output "ecr_repository_url" {
  description = "URL do ECR para docker push"
  value       = module.ecr.repository_url
}

output "ecs_cluster_name" {
  value = module.ecs.cluster_name
}

output "ecs_service_name" {
  value = module.ecs.service_name
}

output "rds_proxy_endpoint" {
  description = "Host do banco que a aplicação deve usar"
  value       = module.rds.db_proxy_endpoint
}

output "db_secret_arn" {
  description = "ARN do Secret para referência em outros sistemas"
  value       = module.secrets.db_secret_arn
  sensitive   = true
}
