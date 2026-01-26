################################
# LOCAL VARIABLES - common across this service - prefer to have them in the tfvars per environment
# for future containers, find and replace 'proxy_router_' with appropriate prefix and update local vars
################################

# ################################
# # OUTPUTS 
# ################################
output "CI_AWS_ECS_CLUSTER_PREFIX" { value = var.proxy_ecs["create"] ? "ecs-${var.proxy_ecs["name"]}" : null }

################################
# ECS CLUSTER, SERVICE & TASK 
################################
# Define ECS Cluster with Fargate as default provider 
resource "aws_ecs_cluster" "proxy_router_use1_1" {
  count    = var.proxy_ecs["create"] ? 1 : 0
  provider = aws.use1
  name     = "ecs-${var.proxy_ecs["name"]}-${substr(var.account_shortname, 8, 3)}-${var.region_shortname}"
  configuration {
    execute_command_configuration {
      kms_key_id = "arn:aws:kms:${var.default_region}:${var.account_number}:alias/foundation-cmk-eks"
      logging    = "OVERRIDE"
      log_configuration {
        cloud_watch_encryption_enabled = false
        cloud_watch_log_group_name     = aws_cloudwatch_log_group.proxy_router[0].name
      }
    }
  }
  setting {
    name  = "containerInsights"
    value = "enabled"
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - ECS Cluster",
      Capability = null,
    },
  )
}

resource "aws_ecs_cluster_capacity_providers" "proxy_router_use1_1" {
  count              = var.proxy_ecs["create"] ? 1 : 0
  provider           = aws.use1
  cluster_name       = aws_ecs_cluster.proxy_router_use1_1[count.index].name
  capacity_providers = ["FARGATE"]
  default_capacity_provider_strategy {
    base              = var.proxy_ecs["task_worker_qty"]
    weight            = 100
    capacity_provider = "FARGATE"
  }
}

###################################################################
######## INTERNAL API ECS Service Discovery Namespace     #########
# 1. Create a public DNS discovery namespace for ECS tasks
resource "aws_service_discovery_public_dns_namespace" "proxy_router_use1_1" {
  count       = var.proxy_ecs["create"] ? 1 : 0
  provider    = aws.use1
  name        = "int.${data.aws_route53_zone.public_default.name}"
  description = "Public DNS discovery namespace for ECS tasks"
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "${local.proxy_service_name_tag} - ECS PubDNS Discovery Namespace",
      Capability = null,
    },
  )
}

# 2. Fetch the NS records for the int.dev.lumerin.io hosted zone
data "aws_route53_zone" "int_subdomain" {
  count        = var.proxy_ecs["create"] ? 1 : 0
  provider     = aws.use1
  name         = aws_service_discovery_public_dns_namespace.proxy_router_use1_1[0].name
  private_zone = false # Set to true if it's private
}

# 3. Create NS record in the parent domain
resource "aws_route53_record" "delegate_subdomain" {
  count    = var.proxy_ecs["create"] ? 1 : 0
  provider = aws.use1
  zone_id  = data.aws_route53_zone.public_lumerin.zone_id
  name     = aws_service_discovery_public_dns_namespace.proxy_router_use1_1[0].name
  type     = "NS"
  ttl      = 300
  records  = data.aws_route53_zone.int_subdomain[0].name_servers
}