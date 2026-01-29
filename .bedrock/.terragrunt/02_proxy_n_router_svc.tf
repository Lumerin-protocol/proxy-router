################################
# for future containers, find and replace 'proxy_router_' with appropriate prefix and update local vars
################################

################################
# OUTPUTS 
################################
output "proxy_router_miner_target" { value = var.proxy_router["create"] ? "stratum+tcp://${aws_route53_record.proxy_router_ext_use1_1[0].name}:${var.proxy_router["svca_alb_port"]}" : null }
output "proxy_router_api_target" { value = var.proxy_router["create"] ? "using AWS VPN: http://${aws_route53_record.proxy_router_int_use1_1[0].name}" : null }

# Variable Output for use in Gitlab CI Vars: 
output "CI_AWS_ECR_REPO" { value = var.proxy_router["create"] ? var.proxy_router["ecr_repo"] : null }
output "CI_AWS_ECR_CLUSTER_REGION" { value = var.proxy_router["create"] ? var.region_shortname : null }

################################
# ECS SERVICE & TASK 
################################

# Define Service (watch for conflict with Gitlab CI/CD)
resource "aws_ecs_service" "proxy_router_use1_1" {
  lifecycle {ignore_changes        = [desired_count, task_definition]}
  count                  = var.proxy_router["create"] ? 1 : 0
  provider               = aws.use1
  name                   = "svc-${var.proxy_router["svc_name"]}-${substr(var.account_shortname, 8, 3)}-${var.region_shortname}"
  cluster                = aws_ecs_cluster.proxy_router_use1_1[count.index].id
  task_definition        = aws_ecs_task_definition.proxy_router_use1_1[count.index].arn
  desired_count          = var.proxy_router["task_worker_qty"]
  launch_type            = "FARGATE"
  propagate_tags         = "SERVICE"
  enable_execute_command = true

  # Deployment configuration: stop-then-start to prevent conflicts
  # Only one task can run at a time (max 100%, min 0%)
  # Grace period: container start + app init + blockchain connection + health check buffer
  deployment_maximum_percent         = 200 # Allow new task while old runs
  deployment_minimum_healthy_percent = 100 # Always keep 1 healthy task
  health_check_grace_period_seconds  = 60

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }
  network_configuration {
    subnets          = [for m in data.aws_subnet.middle : m.id]
    assign_public_ip = false
    security_groups  = [for s in data.aws_security_group.proxy_service_sg : s.id]
  }
  load_balancer {
    target_group_arn = aws_alb_target_group.proxy_router_svca_use1_1[count.index].arn
    container_name   = "${var.proxy_router["cnt_name"]}-container"
    container_port   = var.proxy_router["svca_cnt_port"]
  }
  service_registries {
    registry_arn = aws_service_discovery_service.proxy_router_int_use1_1[0].arn
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - ECS Service",
      Capability = null,
    },
  )
}

# Define Task  
resource "aws_ecs_task_definition" "proxy_router_use1_1" {
  lifecycle {ignore_changes = [container_definitions]}
  count                    = var.proxy_router["create"] ? 1 : 0
  provider                 = aws.use1
  family                   = "tsk-${var.proxy_router["svc_name"]}"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.proxy_router["task_cpu"]
  memory                   = var.proxy_router["task_ram"]
  task_role_arn            = local.titanio_role_arn
  execution_role_arn       = local.titanio_role_arn
  container_definitions = jsonencode([
    {
      name        = "${var.proxy_router["cnt_name"]}-container"
      image       = "${local.titanio_net_ecr}/${var.proxy_router["ecr_repo"]}:${local.proxy_router_image_tag}"
      cpu         = 0
      image_tag   = "latest"
      launch_type = "FARGATE"
      essential   = true
      environment = [
        { "name" : "MULTICALL_ADDRESS", "value" : "0xcA11bde05977b3631167028862bE2a173976CA11" },
        { "name" : "FUTURES_ADDRESS", "value" : var.futures_address },
        # { "name" : "FUTURES_VALIDATOR_URL_OVERRIDE", "value" : var.futures_validator_url_override },
        { "name" : "CLONE_FACTORY_ADDRESS", "value" : var.clone_factory_address },
        { "name" : "VALIDATOR_REGISTRY_ADDRESS", "value" : var.validator_registry_address },
        { "name" : "POOL_ADDRESS", "value" : var.proxy_router["pool_address"] },
        { "name" : "WEB_ADDRESS", "value" : var.proxy_router["web_address"] },
        { "name" : "WEB_PUBLIC_URL", "value" : var.proxy_router["web_public_url"] },
        { "name" : "LOG_IS_PROD", "value" : "true" },
        { "name" : "LOG_LEVEL", "value" : "debug" },
        { "name" : "LOG_JSON", "value" : "true" },
        { "name" : "ENVIRONMENT", "value" : "production" },
        { "name" : "HASHRATE_SHARE_TIMEOUT", "value" : "20m" },
        # OPtional Values        
        # { "name": "ETH_NODE_LEGACY_TX", "value": "" },
        # { "name": "HASHRATE_CYCLE_DURATION", "value": "" },
        # { "name": "HASHRATE_VALIDATION_START_TIMEOUT", "value": "" },
        # { "name": "HASHRATE_ERROR_THRESHOLD", "value": "" },
        # { "name": "HASHRATE_ERROR_TIMEOUT", "value": "" },
        # { "name": "HASHRATE_PEER_VALIDATION_INTERVAL", "value": "" },
        # { "name": "CONTRACT_MNEMONIC", "value": "" },
        # { "name": "WALLET_PRIVATE_KEY", "value": "" },
        # { "name": "MINER_VETTING_DURATION", "value": "" },
        # { "name": "MINER_SHARE_TIMEOUT", "value": "" },
        # { "name": "LOG_COLOR", "value": "" },
        # { "name": "LOG_JSON", "value": "" },
        # { "name": "LOG_LEVEL_APP", "value": "" },
        # { "name": "LOG_LEVEL_CONNECTION", "value": "" },
        # { "name": "LOG_LEVEL_PROXY", "value": "" },
        # { "name": "LOG_LEVEL_SCHEDULER", "value": "" },
        # { "name": "LOG_LEVEL_CONTRACT", "value": "" },
        # { "name": "POOL_CONN_TIMEOUT", "value": "" },
        # { "name": "PROXY_ADDRESS", "value": "" },
        # { "name": "SYS_ENABLE", "value": "" },
        # { "name": "SYS_LOCAL_PORT_RANGE", "value": "" },
        # { "name": "SYS_NET_DEV_MAX_BACKLOG", "value": "" },
        # { "name": "SYS_RLIMIT_HARD", "value": "" },
        # { "name": "SYS_RLIMIT_SOFT", "value": "" },
        # { "name": "SYS_SOMAXCONN", "value": "" },
        # { "name": "SYS_TCP_MAX_SYN_BACKLOG", "value": "" }
      ]
      secrets = [
        { "name" : "WALLET_PRIVATE_KEY", "valueFrom" : "${aws_secretsmanager_secret.proxy_router[0].arn}:wallet_private_key::" },
        { "name" : "ETH_NODE_ADDRESS", "valueFrom" : "${aws_secretsmanager_secret.proxy_router[0].arn}:eth_node_address::" },
        { "name" : "FUTURES_SUBGRAPH_URL", "valueFrom" : "${aws_secretsmanager_secret.proxy_router[0].arn}:futures_subgraph_url::" }
      ]
      portMappings = [
        {
          name          = "proxy_svca"
          containerPort = tonumber(var.proxy_router["svca_cnt_port"])
          hostPort      = tonumber(var.proxy_router["svca_hst_port"])
          protocol      = "tcp" #var.proxy_router["svca_protocol"]

        },
        {
          name          = "proxy_svcb"
          containerPort = tonumber(var.proxy_router["svcb_cnt_port"])
          hostPort      = tonumber(var.proxy_router["svcb_hst_port"])
          protocol      = "tcp" #var.proxy_router["svcb_protocol"]
        }
      ]
      systemControls = []
      ulimits = [
        {
          name      = "nofile"
          softLimit = 15000
          hardLimit = 15000
        }
      ]
      volumesFrom = []
      mountPoints = []
      logConfiguration = {
        logDriver : "awslogs",
        options : {
          awslogs-create-group : "true",
          awslogs-group : aws_cloudwatch_log_group.proxy_router[0].name,
          awslogs-region : var.default_region,
          awslogs-stream-prefix : "${var.proxy_router["svc_name"]}-tsk"
        }
      }
    },
  ])
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - ECS Task",
      Capability = null,
    },
  )
}

################################
# APPLICATION LOAD BALANCER, LISTENER, TARGET GROUP AND WAF 
################################

# EXTERNAL ALB (Internet facing, all Edge Subnets, security group)
resource "aws_alb" "proxy_router_ext_use1_1" {
  count                      = var.proxy_router["create"] ? 1 : 0
  provider                   = aws.use1
  name                       = "alb-${var.proxy_router["svc_name"]}-ext-${var.region_shortname}"
  internal                   = false
  load_balancer_type         = "network"
  subnets                    = [for e in data.aws_subnet.edge : e.id]
  enable_deletion_protection = false
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - NLB",
      Capability = null,
    },
  )
}

############# 7301 Inbound from external  #############
# Create an external tcp/7301 listener on the NLB 
resource "aws_alb_listener" "proxy_router_ext_svca_use1_1" {
  count             = var.proxy_router["create"] ? 1 : 0
  provider          = aws.use1
  load_balancer_arn = aws_alb.proxy_router_ext_use1_1[0].arn
  port              = var.proxy_router["svca_alb_port"]
  protocol          = var.proxy_router["svca_protocol"]
  default_action {
    type             = "forward"
    target_group_arn = aws_alb_target_group.proxy_router_svca_use1_1[0].arn
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - NLB ${var.proxy_router["svca_alb_port"]} External Listener",
      Capability = null,
    },
  )
}

# NLB Internal Target group  (TCP 3333, IP) 
resource "aws_alb_target_group" "proxy_router_svca_use1_1" {
  count                  = var.proxy_router["create"] ? 1 : 0
  provider               = aws.use1
  name                   = "alb-tg-${var.proxy_router["svc_name"]}-${var.proxy_router["svca_hst_port"]}-use1"
  port                   = var.proxy_router["svca_hst_port"]
  protocol               = var.proxy_router["svca_protocol"]
  vpc_id                 = data.aws_vpc.use1_1.id
  target_type            = "ip"
  preserve_client_ip     = true
  connection_termination = true
  deregistration_delay   = 0 # Immediate cutover - miners handle reconnection
  target_health_state {
    enable_unhealthy_connection_termination = true
  }
  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 2
    interval            = 5
    timeout             = 2
    port                = var.proxy_router["svca_hst_port"]
    protocol            = var.proxy_router["svca_protocol"]
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - NLB ${var.proxy_router["svca_hst_port"]} Internal TG ",
      Capability = null,
    },
  )
}

# Define Route53 Alias to load balancer
resource "aws_route53_record" "proxy_router_ext_use1_1" {
  count    = var.proxy_router["create"] ? 1 : 0
  provider = aws.special-dns
  zone_id  = var.account_lifecycle == "prd" ? data.aws_route53_zone.public_default_root.zone_id : data.aws_route53_zone.public_default.zone_id
  name     = var.account_lifecycle == "prd" ? "${var.proxy_router["dns_alb"]}.${data.aws_route53_zone.public_default_root.name}" : "${var.proxy_router["dns_alb"]}.${data.aws_route53_zone.public_default.name}"
  type     = "A"
  alias {
    name                   = aws_alb.proxy_router_ext_use1_1[count.index].dns_name
    zone_id                = aws_alb.proxy_router_ext_use1_1[count.index].zone_id
    evaluate_target_health = true
  }
}

###################################################################
######## INTERNAL API ECS Service Discovery               #########

resource "aws_service_discovery_service" "proxy_router_int_use1_1" {
  count         = var.proxy_router["create"] ? 1 : 0
  provider      = aws.use1
  name          = var.proxy_router["dns_alb_api"] # This should match the service name
  namespace_id  = aws_service_discovery_public_dns_namespace.proxy_router_use1_1[0].id
  force_destroy = true
  dns_config {
    namespace_id = aws_service_discovery_public_dns_namespace.proxy_router_use1_1[0].id
    dns_records {
      type = "A"
      ttl  = 60
    }
  }
  # health_check_custom_config {
  #   failure_threshold = 1
  # }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - ECS Service",
      Capability = null,
    },
  )
}

# Define Route53 Alias to ECS Service Discovery
resource "aws_route53_record" "proxy_router_int_use1_1" {
  count    = var.proxy_router["create"] ? 1 : 0
  provider = aws.special-dns
  #  provider = aws.use1
  zone_id = var.account_lifecycle == "prd" ? data.aws_route53_zone.public_default_root.zone_id : data.aws_route53_zone.public_default.zone_id
  name    = var.account_lifecycle == "prd" ? "${var.proxy_router["dns_alb_api"]}.${data.aws_route53_zone.public_default_root.name}" : "${var.proxy_router["dns_alb_api"]}.${data.aws_route53_zone.public_default.name}"
  type    = "CNAME"
  ttl     = 60
  records = ["${var.proxy_router["dns_alb_api"]}.${aws_service_discovery_public_dns_namespace.proxy_router_use1_1[0].name}"]
}