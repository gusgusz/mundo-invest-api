# ─────────────────────────────────────────────────────────────────────────────
# ECS Module
# Cluster Fargate + Task Definition + Service + Auto Scaling + CloudWatch Logs
# ─────────────────────────────────────────────────────────────────────────────

# ── CloudWatch Log Group ────────────────────────────────────────────────────
resource "aws_cloudwatch_log_group" "app" {
  name              = "/ecs/${var.project}"
  retention_in_days = 30
  tags              = var.tags
}

# ── ECS Cluster ─────────────────────────────────────────────────────────────
resource "aws_ecs_cluster" "main" {
  name = "${var.project}-cluster"

  setting {
    name  = "containerInsights"
    value = "enabled" # métricas detalhadas no CloudWatch
  }

  tags = merge(var.tags, { Name = "${var.project}-cluster" })
}

resource "aws_ecs_cluster_capacity_providers" "main" {
  cluster_name       = aws_ecs_cluster.main.name
  capacity_providers = ["FARGATE", "FARGATE_SPOT"]

  default_capacity_provider_strategy {
    capacity_provider = "FARGATE"
    weight            = 1
    base              = 1 # sempre 1 task FARGATE on-demand (confiabilidade)
  }

  default_capacity_provider_strategy {
    capacity_provider = "FARGATE_SPOT"
    weight            = 2 # tasks adicionais usam SPOT (70% mais barato)
  }
}

# ── IAM Roles ───────────────────────────────────────────────────────────────

# Task Execution Role — permissão do ECS agent para puxar imagem e logs
data "aws_iam_policy_document" "ecs_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "execution" {
  name               = "${var.project}-ecs-execution-role"
  assume_role_policy = data.aws_iam_policy_document.ecs_assume.json
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "execution_managed" {
  role       = aws_iam_role.execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# Permissão extra: ler secrets do Secrets Manager (credenciais injetadas no container)
data "aws_iam_policy_document" "execution_secrets" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [var.db_secret_arn, var.pipefy_secret_arn]
  }
}

resource "aws_iam_role_policy" "execution_secrets" {
  name   = "${var.project}-execution-secrets"
  role   = aws_iam_role.execution.id
  policy = data.aws_iam_policy_document.execution_secrets.json
}

# Task Role — permissão em runtime (o código da aplicação)
resource "aws_iam_role" "task" {
  name               = "${var.project}-ecs-task-role"
  assume_role_policy = data.aws_iam_policy_document.ecs_assume.json
  tags               = var.tags
}

# ── Security Group das Tasks ─────────────────────────────────────────────────
resource "aws_security_group" "ecs_tasks" {
  name        = "${var.project}-sg-ecs"
  description = "Permite tráfego do ALB para as tasks ECS"
  vpc_id      = var.vpc_id

  ingress {
    description     = "HTTP from ALB"
    from_port       = var.container_port
    to_port         = var.container_port
    protocol        = "tcp"
    security_groups = [var.alb_security_group_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"] # necessário para puxar imagem do ECR e chamar Pipefy API
  }

  tags = merge(var.tags, { Name = "${var.project}-sg-ecs" })
}

# ── Task Definition ──────────────────────────────────────────────────────────
resource "aws_ecs_task_definition" "app" {
  family                   = "${var.project}-task"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.task_cpu
  memory                   = var.task_memory
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([
    {
      name      = var.project
      image     = "${var.ecr_repository_url}:${var.image_tag}"
      essential = true

      portMappings = [
        {
          containerPort = var.container_port
          protocol      = "tcp"
        }
      ]

      # Variáveis simples (não-sensíveis)
      environment = [
        { name = "PORT",           value = tostring(var.container_port) },
        { name = "PIPEFY_PIPE_ID", value = var.pipefy_pipe_id },
        { name = "GIN_MODE",       value = "release" }
      ]

      # Variáveis sensíveis — lidas do Secrets Manager (nunca em plaintext)
      secrets = [
        {
          name      = "DB_HOST"
          valueFrom = "${var.db_secret_arn}:host::"
        },
        {
          name      = "DB_PORT"
          valueFrom = "${var.db_secret_arn}:port::"
        },
        {
          name      = "DB_NAME"
          valueFrom = "${var.db_secret_arn}:dbname::"
        },
        {
          name      = "DB_USER"
          valueFrom = "${var.db_secret_arn}:username::"
        },
        {
          name      = "DB_PASSWORD"
          valueFrom = "${var.db_secret_arn}:password::"
        },
        {
          name      = "PIPEFY_TOKEN"
          valueFrom = "${var.pipefy_secret_arn}:token::"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.app.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }

      healthCheck = {
        command     = ["CMD-SHELL", "wget -qO- http://localhost:${var.container_port}/health || exit 1"]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 60
      }
    }
  ])

  tags = var.tags
}

# ── ECS Service ──────────────────────────────────────────────────────────────
resource "aws_ecs_service" "app" {
  name            = "${var.project}-service"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.app.arn
  desired_count   = var.desired_count

  # Deployment rolling - zero downtime
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200

  deployment_circuit_breaker {
    enable   = true  # rollback automático se o deploy falhar
    rollback = true
  }

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = [aws_security_group.ecs_tasks.id]
    assign_public_ip = false # tasks ficam em subnet privada → NAT Gateway
  }

  load_balancer {
    target_group_arn = var.target_group_arn
    container_name   = var.project
    container_port   = var.container_port
  }

  # Ignora mudanças de imagem — o CI/CD atualiza via aws ecs update-service
  lifecycle {
    ignore_changes = [task_definition, desired_count]
  }

  tags = var.tags
}

# ── Auto Scaling ─────────────────────────────────────────────────────────────
resource "aws_appautoscaling_target" "ecs" {
  max_capacity       = var.max_capacity
  min_capacity       = var.min_capacity
  resource_id        = "service/${aws_ecs_cluster.main.name}/${aws_ecs_service.app.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

# Scale out quando CPU > 70%
resource "aws_appautoscaling_policy" "cpu" {
  name               = "${var.project}-scale-cpu"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs.service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = 70.0
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}

# Scale out quando memória > 80%
resource "aws_appautoscaling_policy" "memory" {
  name               = "${var.project}-scale-memory"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs.service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = 80.0
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
  }
}
