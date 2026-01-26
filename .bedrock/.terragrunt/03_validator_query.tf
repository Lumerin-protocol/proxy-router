# Defines the AWS resources to run a Lambda Python function to call Foreman API for specific address 

# Create zip file when Python file changes
resource "null_resource" "validator_zip" {
  triggers = {
    python_file = filemd5("03_validator_query.py")
  }

  provisioner "local-exec" {
    command = "zip -j 03_validator_query.zip 03_validator_query.py"
  }
}

##### Define Lambda Function #####
resource "aws_lambda_function" "validator_lambda" {
  count            = var.validator_query_create ? 1 : 0
  provider         = aws.use1
  function_name    = "${element(var.validator_query["name"], 0)}-lambda"
  role             = aws_iam_role.lumerin_monitoring_lambda_role[0].arn
  runtime          = "python3.13"
  handler          = "03_validator_query.lambda_handler"
  timeout          = 180
  memory_size      = 256
  publish          = true
  filename         = "03_validator_query.zip"         # Replace with the actual ZIP file name of your Lambda code
  source_code_hash = filemd5("03_validator_query.py") # Use Python file hash to trigger updates
  depends_on       = [null_resource.validator_zip]

  vpc_config {
    subnet_ids         = [for n in data.aws_subnet.middle : n.id]
    security_group_ids = [for s in data.aws_security_group.proxy_query : s.id]
  }
  environment {
    variables = {
      # API_URL      = var.validator_api
      API_URL      = "${aws_route53_record.proxy_validator_int_use1_1[0].name}:${var.proxy_validator["svcb_cnt_port"]}"
      ETH_API_KEY  = var.eth_api_key
      ETH_CHAIN    = var.eth_chain
      CW_NAMESPACE = element(var.validator_query["cw_namespace"], 0)
      CW_METRIC1   = element(var.validator_query["cw_metric1"], 0)
      CW_METRIC2   = element(var.validator_query["cw_metric2"], 0)
      CW_METRIC3   = element(var.validator_query["cw_metric3"], 0)
      CW_METRIC4   = element(var.validator_query["cw_metric4"], 0)
      CW_METRIC5   = element(var.validator_query["cw_metric5"], 0)
      CW_METRIC6   = element(var.validator_query["cw_metric6"], 0)
      CW_METRIC7   = element(var.validator_query["cw_metric7"], 0)
      CW_METRIC8   = element(var.validator_query["cw_metric8"], 0)
      CW_METRIC9   = element(var.validator_query["cw_metric9"], 0)
      CW_METRIC10  = element(var.validator_query["cw_metric10"], 0)
      CW_METRIC11  = element(var.validator_query["cw_metric11"], 0)
      CW_METRIC12  = element(var.validator_query["cw_metric12"], 0)
      CW_METRIC13  = element(var.validator_query["cw_metric13"], 0)
    }
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "${element(var.validator_query["name"], 0)} - Lambda Function",
      Application = "${element(var.validator_query["name"], 0)}"
    }
  )
}


##### Attach cloudwatch event to lambda #####
resource "aws_cloudwatch_event_target" "validator_lambda" {
  count     = var.validator_query_create ? 1 : 0
  provider  = aws.use1
  rule      = aws_cloudwatch_event_rule.lumerin_monitoring_schedule_event[0].name
  target_id = "${element(var.validator_query["name"], 0)}-lambda-target"
  arn       = aws_lambda_function.validator_lambda[0].arn
}

resource "aws_lambda_permission" "validator_allow_cloudwatch" {
  count         = var.validator_query_create ? 1 : 0
  provider      = aws.use1
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.validator_lambda[0].function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.lumerin_monitoring_schedule_event[0].arn
  # qualifier     = aws_lambda_alias.test_alias.name
}