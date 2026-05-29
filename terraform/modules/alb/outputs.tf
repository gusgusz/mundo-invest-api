output "alb_dns_name" {
  description = "DNS público do ALB — aponte seu domínio aqui"
  value       = aws_lb.main.dns_name
}

output "alb_zone_id" {
  description = "Hosted Zone ID do ALB (para Route53 alias record)"
  value       = aws_lb.main.zone_id
}

output "alb_arn" {
  value = aws_lb.main.arn
}

output "target_group_arn" {
  value = aws_lb_target_group.app.arn
}

output "alb_security_group_id" {
  value = aws_security_group.alb.id
}
