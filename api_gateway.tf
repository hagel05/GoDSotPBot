# api_gateway.tf
resource "aws_api_gateway_rest_api" "dsotp_bot_api" {
  name        = "dsotp_bot_api"
  description = "API for Slack command handler Lambda function"
}

resource "aws_api_gateway_resource" "dsotp_bot_resource" {
  rest_api_id = aws_api_gateway_rest_api.dsotp_bot_api.id
  parent_id   = aws_api_gateway_rest_api.dsotp_bot_api.root_resource_id
  path_part   = "slack"
}

resource "aws_api_gateway_method" "slack_command_method" {
  rest_api_id   = aws_api_gateway_rest_api.dsotp_bot_api.id
  resource_id   = aws_api_gateway_resource.dsotp_bot_resource.id
  http_method   = "POST"
  authorization = "NONE"
}

# Define the integration between the API Gateway and the Lambda function
resource "aws_api_gateway_integration" "lambda_api_integration" {
  rest_api_id             = aws_api_gateway_rest_api.dsotp_bot_api.id
  resource_id             = aws_api_gateway_resource.dsotp_bot_resource.id
  http_method             = aws_api_gateway_method.slack_command_method.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.go_dsotp_bot_lambda.invoke_arn
}

# Deploy the API
resource "aws_api_gateway_deployment" "dsotp_bot_api_deployment" {
  depends_on = [aws_api_gateway_integration.lambda_api_integration]
  rest_api_id = aws_api_gateway_rest_api.dsotp_bot_api.id
  stage_name  = "prod"
}
