################################################################################
# ECS TASK EXECUTION ROLE SECRETS POLICY
# Allows ECS tasks to pull secrets from Secrets Manager at runtime
# Created as a managed policy (not inline) to match morpheus-router pattern
################################################################################

# Managed policy to allow ECS task execution role to read secrets
resource "aws_iam_policy" "proxy_router_secrets_access" {
  count       = var.proxy_router["create"] ? 1 : 0
  name        = "proxy-router-${var.account_lifecycle}-secrets-policy"
  description = "Allows ECS tasks to read proxy-router and proxy-validator secrets"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "secretsmanager:GetSecretValue"
        Resource = compact([
          var.proxy_router["create"] ? aws_secretsmanager_secret.proxy_router[0].arn : "",
          var.proxy_validator["create"] ? aws_secretsmanager_secret.proxy_validator[0].arn : ""
        ])
      }
    ]
  })
}

# Attach the managed policy to the bedrock-foundation-role
resource "aws_iam_role_policy_attachment" "proxy_router_secrets_access" {
  count      = var.proxy_router["create"] ? 1 : 0
  role       = element(split("/", local.titanio_role_arn), length(split("/", local.titanio_role_arn)) - 1)
  policy_arn = aws_iam_policy.proxy_router_secrets_access[0].arn
}

