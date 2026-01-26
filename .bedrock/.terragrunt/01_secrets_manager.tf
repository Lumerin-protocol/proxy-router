################################################################################
# AWS SECRETS MANAGER
# Stores sensitive configuration for proxy-router and proxy-validator services
# Secrets stored as JSON objects, similar to indexer pattern
################################################################################

################################
# Proxy Router Secret (combined)
################################
resource "aws_secretsmanager_secret" "proxy_router" {
  count       = var.proxy_router["create"] ? 1 : 0
  provider    = aws.use1
  name        = "${var.account_lifecycle}-proxy-router-config"
  description = "Proxy Router configuration for ${var.account_lifecycle}"

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "Proxy Router Config - ${upper(var.account_lifecycle)}"
      Capability = "Secrets Management"
    }
  )
}

resource "aws_secretsmanager_secret_version" "proxy_router" {
  count     = var.proxy_router["create"] ? 1 : 0
  secret_id = aws_secretsmanager_secret.proxy_router[0].id
  secret_string = jsonencode({
    wallet_private_key = var.proxy_wallet_private_key
    eth_node_address   = var.proxy_eth_node_address
  })
}

################################
# Proxy Validator Secret (combined)
################################
resource "aws_secretsmanager_secret" "proxy_validator" {
  count       = var.proxy_validator["create"] ? 1 : 0
  provider    = aws.use1
  name        = "${var.account_lifecycle}-proxy-validator-config"
  description = "Proxy Validator configuration for ${var.account_lifecycle}"

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "Proxy Validator Config - ${upper(var.account_lifecycle)}"
      Capability = "Secrets Management"
    }
  )
}

resource "aws_secretsmanager_secret_version" "proxy_validator" {
  count     = var.proxy_validator["create"] ? 1 : 0
  secret_id = aws_secretsmanager_secret.proxy_validator[0].id
  secret_string = jsonencode({
    wallet_private_key = var.validator_wallet_private_key
    eth_node_address   = var.validator_eth_node_address
  })
}

################################
# Outputs
################################
output "proxy_router_secret_arn" {
  value       = var.proxy_router["create"] ? aws_secretsmanager_secret.proxy_router[0].arn : null
  description = "ARN of proxy-router secret"
}

output "proxy_validator_secret_arn" {
  value       = var.proxy_validator["create"] ? aws_secretsmanager_secret.proxy_validator[0].arn : null
  description = "ARN of proxy-validator secret"
}

