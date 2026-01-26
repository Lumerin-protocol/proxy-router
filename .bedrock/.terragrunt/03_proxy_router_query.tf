# Defines the AWS resources to run a Lambda Python function to call Foreman API for specific address 


# Create zip file when Python file changes
resource "null_resource" "proxy_router_zip" {
  triggers = {
    python_file = filemd5("03_proxy_router_query.py")
  }

  provisioner "local-exec" {
    command = "zip -j 03_proxy_router_query.zip 03_proxy_router_query.py"
  }
}

##### Define Lambda Function #####
resource "aws_lambda_function" "proxy_router_lambda" {
  count            = var.proxy_router_query_create ? 1 : 0
  provider         = aws.use1
  function_name    = "${element(var.proxy_router_query["name"], 0)}-lambda"
  role             = aws_iam_role.lumerin_monitoring_lambda_role[0].arn
  runtime          = "python3.13"
  handler          = "03_proxy_router_query.lambda_handler"
  timeout          = 180
  memory_size      = 256
  publish          = true
  filename         = "03_proxy_router_query.zip"         # Replace with the actual ZIP file name of your Lambda code
  source_code_hash = filemd5("03_proxy_router_query.py") # Use Python file hash to trigger updates
  depends_on       = [null_resource.proxy_router_zip]

  vpc_config {
    subnet_ids         = [for n in data.aws_subnet.middle : n.id]
    security_group_ids = [for s in data.aws_security_group.proxy_query : s.id]
  }
  environment {
    variables = {
      # API_URL      = var.proxy_router_api
      API_URL        = "${aws_route53_record.proxy_router_int_use1_1[0].name}:${var.proxy_router["svcb_cnt_port"]}"
      ETH_API_KEY    = var.eth_api_key
      ETH_CHAIN      = var.eth_chain
      ORACLE_ADDRESS = var.oracle_address
      CW_NAMESPACE   = element(var.proxy_router_query["cw_namespace"], 0)
      CW_METRIC1     = element(var.proxy_router_query["cw_metric1"], 0)
      CW_METRIC2     = element(var.proxy_router_query["cw_metric2"], 0)
      CW_METRIC3     = element(var.proxy_router_query["cw_metric3"], 0)
      CW_METRIC4     = element(var.proxy_router_query["cw_metric4"], 0)
      CW_METRIC5     = element(var.proxy_router_query["cw_metric5"], 0)
      CW_METRIC6     = element(var.proxy_router_query["cw_metric6"], 0)
      CW_METRIC7     = element(var.proxy_router_query["cw_metric7"], 0)
      CW_METRIC8     = element(var.proxy_router_query["cw_metric8"], 0)
      CW_METRIC9     = element(var.proxy_router_query["cw_metric9"], 0)
      CW_METRIC10    = element(var.proxy_router_query["cw_metric10"], 0)
      CW_METRIC11    = element(var.proxy_router_query["cw_metric11"], 0)
      CW_METRIC12    = element(var.proxy_router_query["cw_metric12"], 0)
      CW_METRIC13    = element(var.proxy_router_query["cw_metric13"], 0)
      CW_METRIC14    = element(var.proxy_router_query["cw_metric14"], 0)
      CW_METRIC15    = element(var.proxy_router_query["cw_metric15"], 0)
      CW_METRIC16    = element(var.proxy_router_query["cw_metric16"], 0)
      CW_METRIC17    = element(var.proxy_router_query["cw_metric17"], 0)
      CW_METRIC18    = element(var.proxy_router_query["cw_metric18"], 0)
      CW_METRIC19    = element(var.proxy_router_query["cw_metric19"], 0)
      CW_METRIC20    = element(var.proxy_router_query["cw_metric20"], 0)
      CW_METRIC21    = element(var.proxy_router_query["cw_metric21"], 0)
      CW_METRIC22    = element(var.proxy_router_query["cw_metric22"], 0)
    }
  }
  tags = merge(
    var.default_tags,
    var.foundation_tags,
    {
      Name        = "${element(var.proxy_router_query["name"], 0)} - Lambda Function",
      Application = "${element(var.proxy_router_query["name"], 0)}"
    }
  )
}

##### Attach cloudwatch event to lambda #####
resource "aws_cloudwatch_event_target" "proxy_router_lambda" {
  count     = var.proxy_router_query_create ? 1 : 0
  provider  = aws.use1
  rule      = aws_cloudwatch_event_rule.lumerin_monitoring_schedule_event[0].name
  target_id = "${element(var.proxy_router_query["name"], 0)}-lambda-target"
  arn       = aws_lambda_function.proxy_router_lambda[0].arn
}

resource "aws_lambda_permission" "proxy_router_allow_cloudwatch" {
  count         = var.proxy_router_query_create ? 1 : 0
  provider      = aws.use1
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.proxy_router_lambda[0].function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.lumerin_monitoring_schedule_event[0].arn
  # qualifier     = aws_lambda_alias.test_alias.name
}