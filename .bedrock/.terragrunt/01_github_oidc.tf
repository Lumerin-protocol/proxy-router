################################################################################
# GITHUB OIDC PROVIDER AND IAM ROLES
# Enables GitHub Actions to deploy to ECS without long-lived credentials
################################################################################

################################
# GitHub OIDC Provider
# Use existing provider if already created (e.g., by proxy-ui-foundation)
################################
data "aws_iam_openid_connect_provider" "github" {
  count = var.proxy_router["create"] ? 1 : 0
  url   = "https://token.actions.githubusercontent.com"
}

# Only create if it doesn't exist
resource "aws_iam_openid_connect_provider" "github" {
  count = var.proxy_router["create"] && length(data.aws_iam_openid_connect_provider.github) == 0 ? 1 : 0
  url   = "https://token.actions.githubusercontent.com"

  client_id_list = [
    "sts.amazonaws.com",
  ]

  thumbprint_list = [
    "6938fd4d98bab03faadb97b34396831e3780aea1", # GitHub Actions OIDC thumbprint (legacy)
    "1b511abead59c6ce207077c0bf0e0043b1382612"  # Current GitHub OIDC thumbprint (as of 2023)
  ]

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "GitHub Actions OIDC Provider"
      Capability = "IAM Federation"
    }
  )
}

# Use whichever exists - data source or newly created resource
locals {
  github_oidc_provider_arn = var.proxy_router["create"] ? (
    length(data.aws_iam_openid_connect_provider.github) > 0 ?
    data.aws_iam_openid_connect_provider.github[0].arn :
    aws_iam_openid_connect_provider.github[0].arn
  ) : null
}

################################
# IAM Role for GitHub Actions Deployment
# Each environment only trusts its corresponding branch
################################
locals {
  # Map environment to GitHub branch
  github_branch = var.account_lifecycle == "prd" ? "main" : var.account_lifecycle
  environment_branch = var.account_lifecycle == "prd" ? "lmn" : var.account_lifecycle
}

data "aws_iam_policy_document" "github_actions_assume_role" {
  count = var.proxy_router["create"] ? 1 : 0

  statement {
    effect = "Allow"

    principals {
      type        = "Federated"
      identifiers = [local.github_oidc_provider_arn]
    }

    actions = ["sts:AssumeRoleWithWebIdentity"]

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    # Trust the environment corresponding to this environment
    # When using GitHub Environments, the sub claim format is: repo:ORG/REPO:environment:ENV_NAME
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:Lumerin-protocol/proxy-router:environment:${local.github_branch}"
      ]
    }
  }
}

resource "aws_iam_role" "github_actions_deploy" {
  count              = var.proxy_router["create"] ? 1 : 0
  name               = "github-actions-proxy-router-deploy-${local.environment_branch}"
  assume_role_policy = data.aws_iam_policy_document.github_actions_assume_role[0].json

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name       = "GitHub Actions Deploy Role - ${upper(local.environment_branch)}"
      Capability = "CI/CD"
    }
  )
}

################################
# IAM Policy for ECS Deployment
################################
data "aws_iam_policy_document" "github_actions_deploy_policy" {
  count = var.proxy_router["create"] ? 1 : 0

  # Note: GitHub Actions does NOT need to read secrets
  # Secrets are pulled at runtime by ECS tasks using the task execution role

  # ECS - Describe and register task definitions
  statement {
    sid    = "ManageTaskDefinitions"
    effect = "Allow"
    actions = [
      "ecs:DescribeTaskDefinition",
      "ecs:RegisterTaskDefinition",
      "ecs:DeregisterTaskDefinition",
      "ecs:ListTaskDefinitions"
    ]
    resources = ["*"]
  }

  # ECS - Update services
  statement {
    sid    = "UpdateECSServices"
    effect = "Allow"
    actions = [
      "ecs:UpdateService",
      "ecs:DescribeServices"
    ]
    resources = [
      "arn:aws:ecs:${var.default_region}:${var.account_number}:service/ecs-${var.proxy_router["ecr_repo"]}-${local.environment_branch}-${var.region_shortname}/*"
    ]
  }

  # IAM - Pass role to ECS tasks
  statement {
    sid    = "PassRoleToECS"
    effect = "Allow"
    actions = [
      "iam:PassRole"
    ]
    resources = [
      local.titanio_role_arn # bedrock-foundation-role used by ECS tasks
    ]
    condition {
      test     = "StringEquals"
      variable = "iam:PassedToService"
      values   = ["ecs-tasks.amazonaws.com"]
    }
  }

  # CloudWatch Logs - For deployment logging
  statement {
    sid    = "CloudWatchLogs"
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = [
      "arn:aws:logs:${var.default_region}:${var.account_number}:log-group:/aws/ecs/${var.proxy_router["svc_name"]}*",
      "arn:aws:logs:${var.default_region}:${var.account_number}:log-group:/aws/ecs/${var.proxy_validator["svc_name"]}*"
    ]
  }
}

resource "aws_iam_role_policy" "github_actions_deploy" {
  count  = var.proxy_router["create"] ? 1 : 0
  name   = "github-actions-deploy-policy"
  role   = aws_iam_role.github_actions_deploy[0].id
  policy = data.aws_iam_policy_document.github_actions_deploy_policy[0].json
}

################################
# Outputs
################################
output "github_actions_role_arn" {
  value       = var.proxy_router["create"] ? aws_iam_role.github_actions_deploy[0].arn : null
  description = "ARN of IAM role for GitHub Actions to assume"
}

output "github_oidc_provider_arn" {
  value       = local.github_oidc_provider_arn
  description = "ARN of GitHub OIDC provider"
}

