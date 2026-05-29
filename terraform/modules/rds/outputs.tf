output "db_instance_endpoint" {
  description = "Endpoint direto da instância RDS (use somente para migrations)"
  value       = aws_db_instance.main.endpoint
}

output "db_proxy_endpoint" {
  description = "Endpoint do RDS Proxy — use este na aplicação"
  value       = aws_db_proxy.main.endpoint
}

output "db_security_group_id" {
  value = aws_security_group.rds.id
}

output "db_instance_id" {
  value = aws_db_instance.main.id
}
