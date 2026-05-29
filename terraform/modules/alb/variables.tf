variable "project"                  { type = string }
variable "vpc_id"                   { type = string }
variable "public_subnet_ids"        { type = list(string) }
variable "container_port"           { type = number; default = 8080 }
variable "acm_certificate_arn"      { type = string; description = "ARN do certificado TLS no ACM" }
variable "enable_deletion_protection" { type = bool; default = true }
variable "access_logs_bucket"       { type = string; default = ""; description = "S3 bucket para access logs do ALB (deixe vazio para desabilitar)" }
variable "tags"                     { type = map(string); default = {} }
