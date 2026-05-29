terraform {
  required_version = ">= 1.7"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.50"
    }
  }

  # Backend S3 + DynamoDB para state compartilhado e lock
  # Descomente e preencha após criar o bucket manualmente (bootstrap único)
  # backend "s3" {
  #   bucket         = "mundo-invest-terraform-state"
  #   key            = "production/terraform.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "mundo-invest-terraform-lock"
  #   encrypt        = true
  # }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = var.project
      Environment = var.environment
      ManagedBy   = "Terraform"
      Repository  = "github.com/gusgusz/mundo-invest-api"
    }
  }
}
