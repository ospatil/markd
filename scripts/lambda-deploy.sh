#!/usr/bin/env bash
set -euo pipefail

# Deploy or teardown markd Lambda with function URL
# Usage:
#   ./scripts/lambda-deploy.sh deploy
#   ./scripts/lambda-deploy.sh teardown

FUNCTION_NAME="markd"
ROLE_NAME="markd-lambda-role"
REGION="${AWS_REGION:-$(aws configure get region)}"
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"

deploy() {
  echo "==> Building Lambda binary..."
  make build-lambda
  make css js

  echo "==> Packaging..."
  rm -rf /tmp/lambda-pkg /tmp/markd-lambda.zip
  mkdir -p /tmp/lambda-pkg
  cp bin/bootstrap /tmp/lambda-pkg/
  cp -r static /tmp/lambda-pkg/
  cd /tmp/lambda-pkg && zip -qr /tmp/markd-lambda.zip bootstrap static/
  cd - > /dev/null

  echo "==> Creating IAM role..."
  aws iam create-role \
    --role-name "$ROLE_NAME" \
    --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}' \
    --region "$REGION" --no-cli-pager

  aws iam attach-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

  echo "==> Waiting for IAM propagation..."
  sleep 10

  echo "==> Creating Lambda function..."
  aws lambda create-function \
    --function-name "$FUNCTION_NAME" \
    --runtime provided.al2023 \
    --architectures arm64 \
    --handler bootstrap \
    --role "$ROLE_ARN" \
    --zip-file fileb:///tmp/markd-lambda.zip \
    --memory-size 128 \
    --timeout 30 \
    --environment "Variables={DB_PATH=/tmp/markd.db}" \
    --region "$REGION" --no-cli-pager

  echo "==> Waiting for function to be active..."
  aws lambda wait function-active-v2 --function-name "$FUNCTION_NAME" --region "$REGION"

  echo "==> Creating function URL..."
  URL=$(aws lambda create-function-url-config \
    --function-name "$FUNCTION_NAME" \
    --auth-type NONE \
    --region "$REGION" \
    --query FunctionUrl --output text)

  aws lambda add-permission \
    --function-name "$FUNCTION_NAME" \
    --action lambda:InvokeFunctionUrl \
    --principal "*" \
    --function-url-auth-type NONE \
    --statement-id FunctionURLAllowPublicAccess \
    --region "$REGION" --no-cli-pager

  echo ""
  echo "==> Deployed! URL: ${URL}"
}

teardown() {
  echo "==> Deleting function URL..."
  aws lambda delete-function-url-config \
    --function-name "$FUNCTION_NAME" \
    --region "$REGION" 2>/dev/null || true

  echo "==> Deleting Lambda function..."
  aws lambda delete-function \
    --function-name "$FUNCTION_NAME" \
    --region "$REGION" 2>/dev/null || true

  echo "==> Deleting IAM role..."
  aws iam detach-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole 2>/dev/null || true

  aws iam delete-role \
    --role-name "$ROLE_NAME" 2>/dev/null || true

  echo "==> Cleaned up."
}

case "${1:-}" in
  deploy)   deploy ;;
  teardown) teardown ;;
  *)        echo "Usage: $0 {deploy|teardown}" && exit 1 ;;
esac
