################################
# LOCAL VARIABLES - common across multiple services and environments
################################
locals {
  target_domain                       = "lumerin.io"
  titanio_net_ecr                     = "ghcr.io/lumerin-protocol"
  titanio_role_arn                    = "arn:aws:iam::${var.account_number}:role/system/bedrock-foundation-role"
  x_custom_header_bypass              = var.x_custom_header_bypass #"P4fVAfRcwjaiyrcepvf4PDZW"
  cloudwatch_log_group_name           = "bedrock-${substr(var.account_shortname, 8, 3)}-proxy-router-log-group"
  cloudwatch_validator_log_group_name = "bedrock-${substr(var.account_shortname, 8, 3)}-proxy-validator-log-group"
  cloudwatch_event_retention          = 90
  proxy_service_name_tag              = "Proxy Router V2"
  proxy_service_sg                    = ["outb-all", "weba-all", "lumn-all"]
  proxy_router_int_alb                = ["webu-int"]
  proxy_router_query                  = ["outb-all", "webu-all", "webu-int"]
}