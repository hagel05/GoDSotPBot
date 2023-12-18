locals {
  function_name = "go-dsotp-bot"
  src_path      = "${path.module}/dsotpbot/lambda/src/main/"//${local.function_name}.go"

  binary_name  = local.function_name
  binary_path  = "${path.module}/tf_generated/${local.binary_name}"
  archive_path = "${path.module}/tf_generated/${local.function_name}.zip"
}

output "binary_path" {
  value = local.binary_path
}

// go build -mod=readonly -o ./tf_generated/go-dsotp-bot ./dsotpbot/lambda/src/main/go-dsotp-bot"]