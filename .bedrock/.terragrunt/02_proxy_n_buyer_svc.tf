################################
# for future containers, find and replace 'proxy_buyer_' with appropriate prefix and update local vars
################################

################################
# OUTPUTS 
################################
output "proxy_buyer_miner_target" { value = var.proxy_buyer["create"] ? "stratum+tcp://${aws_route53_record.proxy_buyer_ext_use1_1[0].name}:${var.proxy_buyer["svca_alb_port"]}" : null }
output "proxy_buyer_wallet_target" { value = var.proxy_buyer["create"] ? "http://${aws_route53_record.proxy_buyer_int_use1_1[0].name}:${var.proxy_buyer["svcb_alb_port"]}" : null }
# Variable Output for use in Gitlab CI Vars: 
# output "CI_AWS_ECR_REPO" { value = var.proxy_buyer["create"] ? var.proxy_buyer["ecr_repo"] : null }
# output "CI_AWS_ECR_CLUSTER_REGION" { value = var.proxy_buyer["create"] ? var.region_shortname : null }


################################
# ECS SERVICE & TASK 
################################

# Define Service (watch for conflict with Gitlab CI/CD)
resource "aws_ecs_service" "proxy_buyer_use1_1" {
  
  lifecycle {ignore_changes = [desired_count, task_definition]}

  count                  = var.proxy_buyer["create"] ? 1 : 0
  provider               = aws.use1
  name                   = "svc-${var.proxy_buyer["svc_name"]}-${substr(var.account_shortname, 8, 3)}-${var.region_shortname}"
  cluster                = aws_ecs_cluster.proxy_router_use1_1[count.index].id
  task_definition        = aws_ecs_task_definition.proxy_buyer_use1_1[count.index].arn
  desired_count          = var.proxy_buyer["task_worker_qty"]
  launch_type            = "FARGATE"
  propagate_tags         = "SERVICE"
  enable_execute_command = true
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
    target_group_arn = aws_alb_target_group.proxy_buyer_svca_use1_1[count.index].arn
    container_name   = "${var.proxy_buyer["cnt_name"]}-container"
    container_port   = var.proxy_buyer["svca_cnt_port"]
  }
  load_balancer {
    target_group_arn = aws_alb_target_group.proxy_buyer_svcb_use1_1[count.index].arn
    container_name   = "${var.proxy_buyer["cnt_name"]}-container"
    container_port   = var.proxy_buyer["svcb_cnt_port"]
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
resource "aws_ecs_task_definition" "proxy_buyer_use1_1" {
  
  lifecycle {ignore_changes = [container_definitions]}
  
  count                    = var.proxy_buyer["create"] ? 1 : 0
  provider                 = aws.use1
  family                   = "tsk-${var.proxy_buyer["svc_name"]}"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.proxy_buyer["task_cpu"]
  memory                   = var.proxy_buyer["task_ram"]
  task_role_arn            = local.titanio_role_arn
  execution_role_arn       = local.titanio_role_arn
  container_definitions = jsonencode([
    {
      name        = "${var.proxy_buyer["cnt_name"]}-container"
      image       = "${local.titanio_net_ecr}/${var.proxy_buyer["ecr_repo"]}:${var.proxy_buyer["image_tag"]}"
      cpu         = 0
      image_tag   = "latest"
      launch_type = "FARGATE"
      essential   = true
      portMappings = [
        {
          containerPort = tonumber(var.proxy_buyer["svca_cnt_port"])
          hostPort      = tonumber(var.proxy_buyer["svca_hst_port"])
          protocol      = var.proxy_buyer["svca_protocol"]
        },
        {
          containerPort = tonumber(var.proxy_buyer["svcb_cnt_port"])
          hostPort      = tonumber(var.proxy_buyer["svcb_hst_port"])
          protocol      = var.proxy_buyer["svcb_protocol"]
        }
      ]
      logConfiguration = {
        logDriver : "awslogs",
        options : {
          awslogs-create-group : "true",
          awslogs-group : aws_cloudwatch_log_group.proxy_router[0].name,
          awslogs-region : var.default_region,
          awslogs-stream-prefix : "${var.proxy_buyer["svc_name"]}-tsk"
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
resource "aws_alb" "proxy_buyer_ext_use1_1" {
  count                      = var.proxy_buyer["create"] ? 1 : 0
  provider                   = aws.use1
  name                       = "alb-${var.proxy_buyer["svc_name"]}-ext-${var.region_shortname}"
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
resource "aws_alb_listener" "proxy_buyer_ext_svca_use1_1" {
  count             = var.proxy_buyer["create"] ? 1 : 0
  provider          = aws.use1
  load_balancer_arn = aws_alb.proxy_buyer_ext_use1_1[0].arn
  port              = var.proxy_buyer["svca_alb_port"]
  protocol          = var.proxy_buyer["svca_protocol"]
  default_action {
    type             = "forward"
    target_group_arn = aws_alb_target_group.proxy_buyer_svca_use1_1[0].arn
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - NLB ${var.proxy_buyer["svca_alb_port"]} External Listener",
      Capability = null,
    },
  )
}

# NLB Internal Target group  (TCP 3333, IP) 
resource "aws_alb_target_group" "proxy_buyer_svca_use1_1" {
  count                  = var.proxy_buyer["create"] ? 1 : 0
  provider               = aws.use1
  name                   = "alb-tg-${var.proxy_buyer["svc_name"]}-${var.proxy_buyer["svca_hst_port"]}-use1"
  port                   = var.proxy_buyer["svca_hst_port"]
  protocol               = var.proxy_buyer["svca_protocol"]
  vpc_id                 = data.aws_vpc.use1_1.id
  target_type            = "ip"
  preserve_client_ip     = true
  connection_termination = true
  deregistration_delay   = 0
  target_health_state {
    enable_unhealthy_connection_termination = true
  }
  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 2
    interval            = 5
    timeout             = 2
    port                = var.proxy_buyer["svca_hst_port"]
    protocol            = var.proxy_buyer["svca_protocol"]
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - NLB ${var.proxy_buyer["svca_hst_port"]} Internal TG ",
      Capability = null,
    },
  )
}

# Define Route53 Alias to load balancer
resource "aws_route53_record" "proxy_buyer_ext_use1_1" {
  count    = var.proxy_buyer["create"] ? 1 : 0
  provider = aws.special-dns
  # provider = aws.use1
  zone_id = var.account_lifecycle == "prd" ? data.aws_route53_zone.public_default_root.zone_id : data.aws_route53_zone.public_default.zone_id
  name    = var.account_lifecycle == "prd" ? "${var.proxy_buyer["dns_alb"]}.${data.aws_route53_zone.public_default_root.name}" : "${var.proxy_buyer["dns_alb"]}.${data.aws_route53_zone.public_default.name}"
  type    = "A"
  alias {
    name                   = aws_alb.proxy_buyer_ext_use1_1[count.index].dns_name
    zone_id                = aws_alb.proxy_buyer_ext_use1_1[count.index].zone_id
    evaluate_target_health = true
  }
}

####################################################################
######### INTERNAL API ALB, Target Group and Route53 Alias #########
# Internal ALB (Internet facing, all Edge Subnets, security group)
resource "aws_alb" "proxy_buyer_int_use1_1" {
  count              = var.proxy_buyer["create"] ? 1 : 0
  provider           = aws.use1
  name               = "alb-${var.proxy_buyer["svc_name"]}-int-${var.region_shortname}"
  internal           = true
  load_balancer_type = "network"
  # security_groups            = [for s in data.aws_security_group.proxy_buyer_int_alb : s.id]
  subnets                    = [for e in data.aws_subnet.middle : e.id]
  enable_deletion_protection = false
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - INTERNAL ALB",
      Capability = null,
    },
  )
}

############# 8080 Inbound from external  #############
# Create an internal http/80 listener on the NLB only open to internal traffic
resource "aws_alb_listener" "proxy_buyer_int_svcb_use1_1" {
  count             = var.proxy_buyer["create"] ? 1 : 0
  provider          = aws.use1
  load_balancer_arn = aws_alb.proxy_buyer_int_use1_1[0].arn
  port              = "80"
  # port              = var.proxy_buyer["svcb_alb_port"]
  protocol = var.proxy_buyer["svcb_protocol"]
  default_action {
    type             = "forward"
    target_group_arn = aws_alb_target_group.proxy_buyer_svcb_use1_1[0].arn
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - ALB ${var.proxy_buyer["svcb_alb_port"]} Internal Listener",
      Capability = null,
    },
  )
}

# NLB Internal Target group  (TCP 8080, IP) 
resource "aws_alb_target_group" "proxy_buyer_svcb_use1_1" {
  count                  = var.proxy_buyer["create"] ? 1 : 0
  provider               = aws.use1
  name                   = "alb-tg-${var.proxy_buyer["svc_name"]}-${var.proxy_buyer["svcb_hst_port"]}-use1"
  port                   = var.proxy_buyer["svcb_hst_port"]
  protocol               = var.proxy_buyer["svcb_protocol"]
  vpc_id                 = data.aws_vpc.use1_1.id
  target_type            = "ip"
  preserve_client_ip     = true
  connection_termination = true
  deregistration_delay   = 0
  target_health_state {
    enable_unhealthy_connection_termination = true
  }
  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 2
    interval            = 5
    timeout             = 2
    port                = var.proxy_buyer["svcb_hst_port"]
    protocol            = var.proxy_buyer["svcb_protocol"]
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - NLB ${var.proxy_buyer["svcb_hst_port"]} Internal TG ",
      Capability = null,
    },
  )
}

# Define Route53 Alias to load balancer
resource "aws_route53_record" "proxy_buyer_int_use1_1" {
  count    = var.proxy_buyer["create"] ? 1 : 0
  provider = aws.special-dns
  #  provider = aws.use1
  zone_id = var.account_lifecycle == "prd" ? data.aws_route53_zone.public_default_root.zone_id : data.aws_route53_zone.public_default.zone_id
  name    = var.account_lifecycle == "prd" ? "${var.proxy_buyer["dns_alb_api"]}.${data.aws_route53_zone.public_default_root.name}" : "${var.proxy_buyer["dns_alb_api"]}.${data.aws_route53_zone.public_default.name}"
  type    = "A"
  alias {
    name                   = aws_alb.proxy_buyer_int_use1_1[count.index].dns_name
    zone_id                = aws_alb.proxy_buyer_int_use1_1[count.index].zone_id
    evaluate_target_health = true
  }
}