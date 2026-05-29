variable "project"              { type = string }
variable "aws_region"           { type = string }
variable "vpc_id"               { type = string }
variable "private_subnet_ids"   { type = list(string) }
variable "alb_security_group_id" { type = string }
variable "target_group_arn"     { type = string }
variable "ecr_repository_url"   { type = string }
variable "image_tag"            { type = string; default = "latest" }
variable "container_port"       { type = number; default = 8080 }
variable "task_cpu"             { type = string; default = "256" }
variable "task_memory"          { type = string; default = "512" }
variable "desired_count"        { type = number; default = 2 }
variable "min_capacity"         { type = number; default = 1 }
variable "max_capacity"         { type = number; default = 10 }
variable "db_secret_arn"        { type = string }
variable "pipefy_secret_arn"    { type = string }
variable "pipefy_pipe_id"       { type = string }
variable "tags"                 { type = map(string); default = {} }
