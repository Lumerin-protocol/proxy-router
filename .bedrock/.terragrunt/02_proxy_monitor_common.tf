##### Retrieve the Account's CMK for Secrets Manager #####
data "aws_kms_alias" "secretsmanager" {
  provider = aws.use1
  name     = "alias/foundation-cmk-secretsmanager"
}

##### Define Lambda IAM Role  #####
# Create if at least one of the queries that need to execute lambda are true
resource "aws_iam_role" "lumerin_monitoring_lambda_role" {
  count              = var.proxy_router_query_create || var.validator_query_create || var.financials_query_create || var.wallet_monitor_query_create ? 1 : 0
  provider           = aws.use1
  name               = "lmr-monitoring-role"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "lmr-monitoring - IAM Role",
      Application = "lmr-monitoring"
    }
  )
}

##### Define Lambda IAM Policy #####
resource "aws_iam_policy" "lumerin_monitoring_lambda_policy" {
  count       = var.proxy_router_query_create || var.validator_query_create || var.financials_query_create || var.wallet_monitor_query_create ? 1 : 0
  provider    = aws.use1
  name        = "lumerin-monitoring-policy"
  description = "Policy for the lumerin monitoring lambda function"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "InternetAccess",
      "Effect": "Allow",
      "Action": [
        "ec2:CreateNetworkInterface",
        "ec2:DescribeNetworkInterfaces",
        "ec2:DeleteNetworkInterface"
      ],
      "Resource": "*"
    },
    {
      "Sid": "LogsAccess",
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents", 
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams",
        "logs:PutMetricData"
      ],
      "Resource": "*"
    },
    {
      "Sid": "CloudWatchMetricsAccess",
      "Effect": "Allow",
      "Action": [
        "cloudwatch:PutMetricData"
      ],
      "Resource": "*"
    }, 
    {
      "Sid": "SecretsManagerAccess",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "lambda:InvokeFunction",
      "Resource": "*"
    }
  ]
}
EOF
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "lmr-monitoring - IAM Policy",
      Application = "lmr-monitoring"
    }
  )
}

##### Attach Lambda IAM Policy to Role #####
resource "aws_iam_role_policy_attachment" "lumerin_monitoring_policy_attachment" {
  count      = var.proxy_router_query_create || var.validator_query_create || var.financials_query_create || var.wallet_monitor_query_create ? 1 : 0
  provider   = aws.use1
  role       = aws_iam_role.lumerin_monitoring_lambda_role[0].name
  policy_arn = aws_iam_policy.lumerin_monitoring_lambda_policy[0].arn
}

##### Define Cloudwatch Event Rule #####
resource "aws_cloudwatch_event_rule" "lumerin_monitoring_schedule_event" {
  count               = var.proxy_router_query_create || var.validator_query_create || var.financials_query_create || var.wallet_monitor_query_create ? 1 : 0
  provider            = aws.use1
  name                = "lmr-monitoring-schedule-event"
  description         = "Schedule event to trigger the Lambda function every 5 minutes"
  schedule_expression = var.monitoring_frequency
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "lmr-monitoring - Event Schedule",
      Application = "lmr-monitoring"
    }
  )
}