output "cluster_name" {
  value = aws_ecs_cluster.main.name
}

output "service_name" {
  value = aws_ecs_service.app.name
}

output "task_definition_arn" {
  value = aws_ecs_task_definition.app.arn
}

output "ecs_task_security_group_id" {
  value = aws_security_group.ecs_tasks.id
}

output "log_group_name" {
  value = aws_cloudwatch_log_group.app.name
}
