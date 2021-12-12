resource "aws_lambda_function" "otel" {
  function_name = "OpenTelemetryHandler"

  s3_bucket = aws_s3_bucket.lambda_bucket.id
  s3_key    = aws_s3_bucket_object.lambda_otel.key

  runtime = "go1.x"
  handler = "otel"

  source_code_hash = data.archive_file.lambda_otel.output_base64sha256

  role = aws_iam_role.lambda_exec.arn

   environment {
    variables = {
      LUMIGO_DEBUG = "true"
    }
  }
}

resource "aws_cloudwatch_log_group" "otel" {
  name = "/aws/lambda/${aws_lambda_function.otel.function_name}"

  retention_in_days = 30
}

resource "aws_iam_role" "lambda_exec" {
  name = "serverless_lambda"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Sid    = ""
      Principal = {
        Service = "lambda.amazonaws.com"
      }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_policy" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "lambda_s3_read" {
  name = "lambda_s3_read_policy"
  role = aws_iam_role.lambda_exec.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
            "s3:ListAllMyBuckets"
        ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
EOF
}