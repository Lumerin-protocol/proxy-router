############################################################################################################
# Create the default CloudWatch Log Group resource for proxy_router_ECS Services 
resource "aws_cloudwatch_log_group" "proxy_router" {
  count             = var.proxy_router["create"] ? 1 : 0
  provider          = aws.use1
  name              = local.cloudwatch_log_group_name
  retention_in_days = local.cloudwatch_event_retention
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Capability = "Bedrock Cloudwatch Log Group",
    },
  )
}

############################################################################################################
# Create the Validator CloudWatch Log Group resource for proxy_router_ECS Services 
resource "aws_cloudwatch_log_group" "proxy_validator" {
  count             = var.proxy_validator["create"] ? 1 : 0
  provider          = aws.use1
  name              = local.cloudwatch_validator_log_group_name
  retention_in_days = local.cloudwatch_event_retention
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Capability = "Bedrock Cloudwatch Log Group",
    },
  )
}

############################################################################################################
## IAM and Access permissions for Cloudwatch for Seller and Validtor logging 
# Create the IAM Role
resource "aws_iam_role" "proxy_router" {
  count              = var.proxy_router["create"] ? 1 : 0
  provider           = aws.use1
  name               = "${local.cloudwatch_log_group_name}-ecs-cw-role"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ecs.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Capability = "Bedrock IAM Role",
    },
  )
}

# Create the Inline Policy for the IAM Role
resource "aws_iam_role_policy" "proxy_router" {
  count    = var.proxy_router["create"] ? 1 : 0
  provider = aws.use1
  name     = "${local.cloudwatch_log_group_name}-ecs-cw-policy"
  role     = aws_iam_role.proxy_router[0].id
  policy   = data.aws_iam_policy_document.proxy_router_cloudwatch_log_stream.json
}

# Create the Log Stream Policy JSON data for the CloudWatch Logs Role
data "aws_iam_policy_document" "proxy_router_cloudwatch_log_stream" {
  statement {
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = [
      "arn:aws:logs:${var.default_region}:${var.account_number}:log-group:${local.cloudwatch_log_group_name}:log-stream:*",
      "arn:aws:logs:${var.default_region}:${var.account_number}:log-group:${local.cloudwatch_validator_log_group_name}:log-stream:*",
    ]
  }
}