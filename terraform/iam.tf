module "irsa_s3_backup" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.0"

  role_name = "kv-s3-backup-role"
  
  role_policy_arns = {
    policy = aws_iam_policy.s3_backup.arn
  }

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["default:kv-s3-sa"]
    }
  }
}

resource "aws_iam_policy" "s3_backup" {
  name        = "kv-s3-backup-policy"
  description = "Allows KV nodes to backup SSTables to S3"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action   = ["s3:PutObject", "s3:GetObject"]
        Effect   = "Allow"
        Resource = "arn:aws:s3:::my-kv-backup-bucket/*"
      },
    ]
  })
}
