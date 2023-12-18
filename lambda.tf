// build the binary for the lambda function in a specified path
resource "null_resource" "function_binary" {
  provisioner "local-exec" {
    // command = "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOFLAGS=-trimpath go build -mod=readonly -ldflags='-s -w' -o ${local.binary_path} ${local.src_path}"
    command = <<-EOT
      set GOOS=linux&& set GOARCH=amd64&&go build -mod=readonly -o ${local.binary_path} ${local.src_path}
    EOT
  }
}

// zip the binary, as we can use only zip files to AWS lambda
data "archive_file" "function_archive" {
  depends_on = [null_resource.function_binary]

  type        = "zip"
  source_file = local.binary_path
  output_path = local.archive_path
}

// create the lambda function from zip file
resource "aws_lambda_function" "function" {
  function_name = "go-dsotp-bot"
  description   = "A rewrite of the orginal bot but in GoLang"
  role          = aws_iam_role.lambda.arn
  handler       = local.binary_name
  memory_size   = 128

  filename         = local.archive_path
  source_code_hash = data.archive_file.function_archive.output_base64sha256

  runtime = "go1.x"
}

// create log group in cloudwatch to gather logs of our lambda function
resource "aws_cloudwatch_log_group" "log_group" {
  name              = "/aws/lambda/${aws_lambda_function.function.function_name}"
  retention_in_days = 7
}