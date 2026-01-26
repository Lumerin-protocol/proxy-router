# Wallet Monitor Lambda - Monitors ETH, USDC, and LMR balances for specified wallets
# Publishes metrics to CloudWatch with wallet name dimensions for alerting

##### SNS Topic for Alerts #####
# Use last 3 characters of account_shortname (e.g., titanio-dev → dev, titanio-lmn → lmn)
locals {
  env_suffix = substr(var.account_shortname, -3, 3)
}

data "aws_sns_topic" "wallet_alerts" {
  count    = var.wallet_monitor_query_create ? 1 : 0
  provider = aws.use1
  name     = "titanio-${local.env_suffix}-dev-alerts"
}

# Create zip file when Python file changes
resource "null_resource" "wallet_monitor_zip" {
  triggers = {
    python_file = filemd5("03_wallet_monitor_query.py")
  }

  provisioner "local-exec" {
    command = "zip -j 03_wallet_monitor_query.zip 03_wallet_monitor_query.py"
  }
}

##### Define Lambda Function #####
resource "aws_lambda_function" "wallet_monitor_lambda" {
  count            = var.wallet_monitor_query_create ? 1 : 0
  provider         = aws.use1
  function_name    = "${var.wallet_monitor_query["name"]}-lambda"
  role             = aws_iam_role.lumerin_monitoring_lambda_role[0].arn
  runtime          = "python3.13"
  handler          = "03_wallet_monitor_query.lambda_handler"
  timeout          = 300 # 5 minutes - may need more time for multiple wallets
  memory_size      = 256
  publish          = true
  filename         = "03_wallet_monitor_query.zip"
  source_code_hash = filemd5("03_wallet_monitor_query.py")
  depends_on       = [null_resource.wallet_monitor_zip]

  vpc_config {
    subnet_ids         = [for n in data.aws_subnet.middle : n.id]
    security_group_ids = [for s in data.aws_security_group.proxy_query : s.id]
  }

  environment {
    variables = {
      ETH_CHAIN          = var.eth_chain
      ETH_API_KEY        = var.eth_api_key
      CW_NAMESPACE       = var.wallet_monitor_query["cw_namespace"]
      REGION_NAME        = var.default_region
      WALLETS_TO_WATCH   = jsonencode(var.wallets_to_watch)
      LMR_TOKEN_ADDRESS  = var.wallet_monitor_query["lmr_token_address"]
      USDC_TOKEN_ADDRESS = var.wallet_monitor_query["usdc_token_address"]
    }
  }

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "${var.wallet_monitor_query["name"]} - Lambda Function",
      Application = var.wallet_monitor_query["name"]
    }
  )
}

##### Define Separate CloudWatch Event Rule for Wallet Monitor #####
# This allows a different schedule than the main monitoring (e.g., every 15 minutes)
resource "aws_cloudwatch_event_rule" "wallet_monitor_schedule" {
  count               = var.wallet_monitor_query_create ? 1 : 0
  provider            = aws.use1
  name                = "${var.wallet_monitor_query["name"]}-schedule"
  description         = "Schedule event to trigger the wallet monitor Lambda function"
  schedule_expression = var.wallet_monitor_frequency

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "${var.wallet_monitor_query["name"]} - Event Schedule",
      Application = var.wallet_monitor_query["name"]
    }
  )
}

##### Attach CloudWatch Event to Lambda #####
resource "aws_cloudwatch_event_target" "wallet_monitor_lambda" {
  count     = var.wallet_monitor_query_create ? 1 : 0
  provider  = aws.use1
  rule      = aws_cloudwatch_event_rule.wallet_monitor_schedule[0].name
  target_id = "${var.wallet_monitor_query["name"]}-lambda-target"
  arn       = aws_lambda_function.wallet_monitor_lambda[0].arn
}

resource "aws_lambda_permission" "wallet_monitor_allow_cloudwatch" {
  count         = var.wallet_monitor_query_create ? 1 : 0
  provider      = aws.use1
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.wallet_monitor_lambda[0].function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.wallet_monitor_schedule[0].arn
}

##### CloudWatch Alarms for Low Balances #####
# Create alarms for each monitored wallet when balances drop below thresholds

# ETH Balance Alarm (per wallet)
resource "aws_cloudwatch_metric_alarm" "wallet_eth_low" {
  for_each = var.wallet_monitor_query_create ? {
    for wallet in var.wallets_to_watch : wallet.walletName => wallet
    if lookup(wallet, "eth_alarm_threshold", null) != null
  } : {}

  provider            = aws.use1
  alarm_name          = "wallet-${lower(each.key)}-eth-low"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.wallet_monitor_query["alarm_evaluation_periods"]
  metric_name         = "eth_balance"
  namespace           = var.wallet_monitor_query["cw_namespace"]
  period              = var.wallet_monitor_query["alarm_period"]
  statistic           = "Average"
  threshold           = each.value.eth_alarm_threshold
  treat_missing_data  = "notBreaching"

  alarm_description = <<-EOT
    ${upper(local.env_suffix)} - ${upper(each.key)} - ETH - LOW
    
    Please add ETH to the ${each.value.walletId} wallet to bring it back to ${each.value.eth_alarm_threshold} ETH.
    
    Current threshold: ${each.value.eth_alarm_threshold} ETH
    Wallet Name: ${each.key}
    Wallet Address: ${each.value.walletId}
    Environment: ${local.env_suffix}
  EOT

  alarm_actions = [data.aws_sns_topic.wallet_alerts[0].arn]
  ok_actions    = [data.aws_sns_topic.wallet_alerts[0].arn]

  dimensions = {
    WalletName = each.key
  }

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "Wallet ${each.key} ETH Low Alarm",
      Application = var.wallet_monitor_query["name"]
    }
  )
}

# USDC Balance Alarm (per wallet)
resource "aws_cloudwatch_metric_alarm" "wallet_usdc_low" {
  for_each = var.wallet_monitor_query_create ? {
    for wallet in var.wallets_to_watch : wallet.walletName => wallet
    if lookup(wallet, "usdc_alarm_threshold", null) != null
  } : {}

  provider            = aws.use1
  alarm_name          = "wallet-${lower(each.key)}-usdc-low"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.wallet_monitor_query["alarm_evaluation_periods"]
  metric_name         = "usdc_balance"
  namespace           = var.wallet_monitor_query["cw_namespace"]
  period              = var.wallet_monitor_query["alarm_period"]
  statistic           = "Average"
  threshold           = each.value.usdc_alarm_threshold
  treat_missing_data  = "notBreaching"

  alarm_description = <<-EOT
    ${upper(local.env_suffix)} - ${upper(each.key)} - USDC - LOW
    
    Please add USDC to the ${each.value.walletId} wallet to bring it back to ${each.value.usdc_alarm_threshold} USDC.
    
    Current threshold: ${each.value.usdc_alarm_threshold} USDC
    Wallet Name: ${each.key}
    Wallet Address: ${each.value.walletId}
    Environment: ${local.env_suffix}
  EOT

  alarm_actions = [data.aws_sns_topic.wallet_alerts[0].arn]
  ok_actions    = [data.aws_sns_topic.wallet_alerts[0].arn]

  dimensions = {
    WalletName = each.key
  }

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "Wallet ${each.key} USDC Low Alarm",
      Application = var.wallet_monitor_query["name"]
    }
  )
}

# LMR Balance Alarm (per wallet)
resource "aws_cloudwatch_metric_alarm" "wallet_lmr_low" {
  for_each = var.wallet_monitor_query_create ? {
    for wallet in var.wallets_to_watch : wallet.walletName => wallet
    if lookup(wallet, "lmr_alarm_threshold", null) != null
  } : {}

  provider            = aws.use1
  alarm_name          = "wallet-${lower(each.key)}-lmr-low"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.wallet_monitor_query["alarm_evaluation_periods"]
  metric_name         = "lmr_balance"
  namespace           = var.wallet_monitor_query["cw_namespace"]
  period              = var.wallet_monitor_query["alarm_period"]
  statistic           = "Average"
  threshold           = each.value.lmr_alarm_threshold
  treat_missing_data  = "notBreaching"

  alarm_description = <<-EOT
    ${upper(local.env_suffix)} - ${upper(each.key)} - LMR TOKEN - LOW
    
    Please add LMR tokens to the ${each.value.walletId} wallet to bring it back to ${each.value.lmr_alarm_threshold} LMR.
    
    Current threshold: ${each.value.lmr_alarm_threshold} LMR
    Wallet Name: ${each.key}
    Wallet Address: ${each.value.walletId}
    Environment: ${local.env_suffix}
  EOT

  alarm_actions = [data.aws_sns_topic.wallet_alerts[0].arn]
  ok_actions    = [data.aws_sns_topic.wallet_alerts[0].arn]

  dimensions = {
    WalletName = each.key
  }

  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "Wallet ${each.key} LMR Low Alarm",
      Application = var.wallet_monitor_query["name"]
    }
  )
}
