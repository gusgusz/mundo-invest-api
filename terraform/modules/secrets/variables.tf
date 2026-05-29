variable "project"             { type = string }
variable "db_username"         { type = string }
variable "db_password"         { type = string; sensitive = true }
variable "db_name"             { type = string }
variable "db_proxy_endpoint"   { type = string }
variable "pipefy_token"        { type = string; sensitive = true }
variable "tags"                { type = map(string); default = {} }
