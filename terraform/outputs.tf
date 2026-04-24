output "rpc_endpoint" {
  description = "Public RPC endpoint"
  value       = "http://${aws_lb.main.dns_name}"
}

output "ecr_repository_url" {
  description = "ECR URL to push your Docker image"
  value       = aws_ecr_repository.app.repository_url
}