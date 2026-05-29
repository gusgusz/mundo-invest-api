variable "repository_name" {
  description = "Nome do repositório ECR"
  type        = string
}

variable "tags" {
  type    = map(string)
  default = {}
}
