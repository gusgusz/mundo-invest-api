variable "project"             { type = string }
variable "vpc_id"              { type = string }
variable "vpc_cidr"            { type = string }
variable "private_subnet_ids"  { type = list(string) }
variable "db_name"             { type = string }
variable "db_username"         { type = string }
variable "db_password"         { type = string; sensitive = true }
variable "db_secret_arn"       { type = string; description = "ARN do secret no Secrets Manager com credenciais do DB" }
variable "db_instance_class"   { type = string; default = "db.t3.medium" }
variable "allocated_storage"   { type = number; default = 20 }
variable "max_allocated_storage" { type = number; default = 100 }
variable "multi_az"            { type = bool; default = true }
variable "deletion_protection" { type = bool; default = true }
variable "tags"                { type = map(string); default = {} }
