terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "helixflow-terraform-state"
    key    = "aws-infrastructure.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Environment = var.environment
      Project     = "helixflow"
      ManagedBy   = "terraform"
    }
  }
}

module "vpc" {
  source = "../modules/vpc"

  name = "helixflow-${var.environment}"
  cidr = var.vpc_cidr

  azs             = var.availability_zones
  private_subnets = var.private_subnets
  public_subnets  = var.public_subnets

  enable_nat_gateway = true
  single_nat_gateway = var.environment == "development"

  tags = local.common_tags
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 19.0"

  cluster_name    = "helixflow-${var.environment}"
  cluster_version = "1.28"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  eks_managed_node_groups = {
    general = {
      name = "general"

      instance_types = ["m5.large", "m5.xlarge"]
      min_size       = 1
      max_size       = 10
      desired_size   = 2

      labels = {
        Environment = var.environment
        NodeGroup   = "general"
      }
    }

    gpu = {
      name = "gpu-nodes"

      instance_types = ["g4dn.xlarge", "g4dn.2xlarge"]
      min_size       = 0
      max_size       = 5
      desired_size   = 1

      labels = {
        Environment = var.environment
        NodeGroup   = "gpu"
        GPU         = "enabled"
      }

      taints = [
        {
          key    = "nvidia.com/gpu"
          value  = "present"
          effect = "NoSchedule"
        }
      ]
    }
  }

  node_security_group_additional_rules = {
    ingress_self_all = {
      description = "Node to node all ports/protocols"
      protocol    = "-1"
      from_port   = 0
      to_port     = 0
      type        = "ingress"
      self        = true
    }
  }

  tags = local.common_tags
}

module "rds" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 6.0"

  identifier = "helixflow-${var.environment}"

  engine            = "postgres"
  engine_version    = "15.4"
  instance_class    = "db.r6g.large"
  allocated_storage = 100

  db_name  = "helixflow"
  username = "helixflow_admin"
  port     = "5432"

  vpc_security_group_ids = [aws_security_group.rds.id]
  db_subnet_group_name   = aws_db_subnet_group.helixflow.name

  maintenance_window = "Mon:00:00-Mon:03:00"
  backup_window      = "03:00-06:00"

  backup_retention_period = 30
  skip_final_snapshot     = var.environment == "development"
  deletion_protection     = var.environment == "production"

  performance_insights_enabled = true
  monitoring_interval          = 60

  tags = local.common_tags
}

module "elasticache" {
  source  = "terraform-aws-modules/elasticache/aws"
  version = "~> 1.0"

  cluster_id      = "helixflow-${var.environment}"
  engine          = "redis"
  node_type       = "cache.r6g.large"
  num_cache_nodes = 2
  port            = 6379

  subnet_ids         = module.vpc.private_subnets
  security_group_ids = [aws_security_group.elasticache.id]

  maintenance_window = "sun:05:00-sun:09:00"
  snapshot_window    = "00:00-05:00"

  tags = local.common_tags
}

locals {
  common_tags = {
    Environment = var.environment
    Project     = "helixflow"
    ManagedBy   = "terraform"
  }
}