# Defines the AWS resources to run a Lambda Python function to call Foreman API for specific address 

data "archive_file" "indexer_query" {
  type        = "zip"
  source_file = "${path.module}/03_indexer_query.py"
  output_path = "${path.module}/03_indexer_query.zip"
}

##### Define Lambda Function #####
resource "aws_lambda_function" "indexer_lambda" {
  count            = var.indexer_query_create ? 1 : 0
  provider         = aws.use1
  function_name    = "${element(var.indexer_query["name"], 0)}-lambda"
  role             = aws_iam_role.lumerin_monitoring_lambda_role[0].arn
  runtime          = "python3.13"
  handler          = "03_indexer_query.lambda_handler"
  timeout          = 180
  memory_size      = 256
  publish          = true
  filename         = data.archive_file.indexer_query.output_path
  source_code_hash = data.archive_file.indexer_query.output_base64sha256

  vpc_config {
    subnet_ids         = [for n in data.aws_subnet.middle : n.id]
    security_group_ids = [for s in data.aws_security_group.proxy_query : s.id]
  }
  environment {
    variables = {
      API_URL      = var.account_lifecycle == "prd" ? "indexer.lumerin.io/api" : "indexer.${var.account_shortname}.lumerin.io/api"
      ETH_API_KEY  = var.eth_api_key
      ETH_CHAIN    = var.eth_chain
      CW_NAMESPACE = element(var.indexer_query["cw_namespace"], 0)
      CW_METRIC1   = element(var.indexer_query["cw_metric1"], 0)
      CW_METRIC2   = element(var.indexer_query["cw_metric2"], 0)
    }
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "${element(var.indexer_query["name"], 0)} - Lambda Function",
      Application = "${element(var.indexer_query["name"], 0)}"
    }
  )
}


##### Attach cloudwatch event to lambda #####
resource "aws_cloudwatch_event_target" "indexer_lambda" {
  count     = var.indexer_query_create ? 1 : 0
  provider  = aws.use1
  rule      = aws_cloudwatch_event_rule.lumerin_monitoring_schedule_event[0].name
  target_id = "${element(var.indexer_query["name"], 0)}-lambda-target"
  arn       = aws_lambda_function.indexer_lambda[0].arn
}

resource "aws_lambda_permission" "indexer_allow_cloudwatch" {
  count         = var.indexer_query_create ? 1 : 0
  provider      = aws.use1
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.indexer_lambda[0].function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.lumerin_monitoring_schedule_event[0].arn
  # qualifier     = aws_lambda_alias.test_alias.name
}