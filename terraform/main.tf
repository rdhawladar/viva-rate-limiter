terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  
  backend "s3" {
    bucket = "viva-terraform-state-1754904849"
    region = "ap-southeast-1"
    # Key will be set dynamically per environment
  }
}

locals {
  cluster_name = var.cluster_name != "" ? var.cluster_name : "viva-cluster-${var.environment}"
  
  # Environment-specific configurations
  env_config = {
    dev = {
      node_instance_type = "t3.small"
      desired_capacity   = 1
      max_capacity       = 2
      min_capacity       = 1
    }
    stage = {
      node_instance_type = "t3.medium"
      desired_capacity   = 2
      max_capacity       = 3
      min_capacity       = 2
    }
    prod = {
      node_instance_type = "t3.large"
      desired_capacity   = 3
      max_capacity       = 5
      min_capacity       = 3
    }
  }
  
  config = local.env_config[var.environment]
}

provider "aws" {
  region = var.aws_region
}

# VPC Configuration
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = "viva-vpc-${var.environment}"
  cidr = "10.0.0.0/16"

  azs             = ["${var.aws_region}a", "${var.aws_region}b"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]

  enable_nat_gateway = true
  single_nat_gateway = var.environment == "dev" ? true : false
  enable_dns_hostnames = true

  tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "Environment" = var.environment
  }

  public_subnet_tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "kubernetes.io/role/elb"                      = "1"
  }

  private_subnet_tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "kubernetes.io/role/internal-elb"             = "1"
  }
}

# EKS Cluster
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "19.15.3"

  cluster_name    = local.cluster_name
  cluster_version = "1.30"

  vpc_id                         = module.vpc.vpc_id
  subnet_ids                     = module.vpc.private_subnets
  cluster_endpoint_public_access = true

  eks_managed_node_group_defaults = {
    instance_types = [local.config.node_instance_type]
  }

  eks_managed_node_groups = {
    main = {
      min_size     = local.config.min_capacity
      max_size     = local.config.max_capacity
      desired_size = local.config.desired_capacity

      instance_types = [local.config.node_instance_type]
      
      tags = {
        Environment = var.environment
        Application = "viva-rate-limiter"
      }
    }
  }
}

# ECR Repository (shared across environments)
resource "aws_ecr_repository" "viva_rate_limiter" {
  name                 = "viva-rate-limiter"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
  
  tags = {
    Environment = "shared"
    Application = "viva-rate-limiter"
  }
}

# IAM Role for GitHub Actions
resource "aws_iam_role" "github_actions" {
  name = "github-actions-eks-deploy"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "github_actions_policy" {
  name = "github-actions-eks-policy"
  role = aws_iam_role.github_actions.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "eks:DescribeCluster",
          "eks:ListClusters",
          "ecr:GetAuthorizationToken",
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload"
        ]
        Resource = "*"
      }
    ]
  })
}

data "aws_caller_identity" "current" {}

# Outputs
output "cluster_endpoint" {
  description = "Endpoint for EKS control plane"
  value       = module.eks.cluster_endpoint
}

output "cluster_name" {
  description = "Kubernetes Cluster Name"
  value       = module.eks.cluster_name
}

output "ecr_repository_url" {
  description = "URL of the ECR repository"
  value       = aws_ecr_repository.viva_rate_limiter.repository_url
}

output "configure_kubectl" {
  description = "Configure kubectl command"
  value       = "aws eks --region ${var.aws_region} update-kubeconfig --name ${module.eks.cluster_name}"
}