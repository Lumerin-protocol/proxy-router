############################################################################################################
# Seller Metric Filters for CloudWatch
resource "aws_cloudwatch_log_metric_filter" "seller_warn" {
  count          = var.proxy_router["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Seller-Warn"
  pattern        = "{ $.level = \"warn\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_router[0].name

  metric_transformation {
    name      = "seller_warn"
    namespace = var.proxy_router["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "seller_error" {
  count          = var.proxy_router["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Seller-Error"
  pattern        = "{ $.level = \"error\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_router[0].name

  metric_transformation {
    name      = "seller_error"
    namespace = var.proxy_router["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "seller_lowdiff" {
  count          = var.proxy_router["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Seller-LowDiff"
  pattern        = "{ $.msg = \"*low difficulty share*\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_router[0].name

  metric_transformation {
    name      = "seller_lowdiff"
    namespace = var.proxy_router["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "seller_failedtoconnect" {
  count          = var.proxy_router["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Seller-FailedToConnect"
  pattern        = "{ $.msg = \"*failed to connect*\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_router[0].name

  metric_transformation {
    name      = "seller_failedtoconnect"
    namespace = var.proxy_router["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "seller_invalidshares" {
  count          = var.proxy_router["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Seller-ConsequentInvalidShares"
  pattern        = "{ $.msg = \"*too many consequent invalid shares*\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_router[0].name

  metric_transformation {
    name      = "seller_consequentinvalidshares"
    namespace = var.proxy_router["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "seller_invalid_work" {
  count          = var.proxy_router["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Seller-InvalidWork"
  pattern        = "{ $.msg = \"*invalid-work*\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_router[0].name

  metric_transformation {
    name      = "seller_invalid_work"
    namespace = var.proxy_router["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "seller_job_not_found" {
  count          = var.proxy_router["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Seller-JobNotFound"
  pattern        = "{ $.msg = \"*job not found*\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_router[0].name

  metric_transformation {
    name      = "seller_job_not_found"
    namespace = var.proxy_router["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

# {"L":"INFO","T":"2024-02-19T15:37:50","N":"APP","M":"open files: 9"}

############################################################################################################
# Validator Metric Filters for CloudWatch
resource "aws_cloudwatch_log_metric_filter" "validator_warn" {
  count          = var.proxy_validator["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Validator-Warn"
  pattern        = "{ $.level = \"warn\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_validator[0].name

  metric_transformation {
    name      = "validator_warn"
    namespace = var.proxy_validator["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "validator_error" {
  count          = var.proxy_validator["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "Validator-Error"
  pattern        = "{ $.level = \"error\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_validator[0].name

  metric_transformation {
    name      = "validator_error"
    namespace = var.proxy_validator["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "validator_invalid_work" {
  count          = var.proxy_validator["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "validator-InvalidWork"
  pattern        = "{ $.msg = \"*invalid-work*\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_validator[0].name

  metric_transformation {
    name      = "validator_invalid_work"
    namespace = var.proxy_validator["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "validator_job_not_found" {
  count          = var.proxy_validator["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "validator-JobNotFound"
  pattern        = "{ $.msg = \"*job not found*\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_validator[0].name

  metric_transformation {
    name      = "validator_job_not_found"
    namespace = var.proxy_validator["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}

resource "aws_cloudwatch_log_metric_filter" "validator_contract_cancelled" {
  count          = var.proxy_validator["monitor_metric_filters"] ? 1 : 0
  provider       = aws.use1
  name           = "validator-ContractCancelled"
  pattern        = "{ $.msg = \"*buyer contract closed, with type cancel*\" }"
  log_group_name = aws_cloudwatch_log_group.proxy_validator[0].name

  metric_transformation {
    name      = "validator_contract_cancelled"
    namespace = var.proxy_validator["svc_name"]
    value     = "1"
    unit      = "Count"
  }
}