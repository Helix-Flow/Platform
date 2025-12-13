terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  backend "gcs" {
    bucket = "helixflow-terraform-state"
    prefix = "gcp-infrastructure"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region

  default_labels = {
    environment = var.environment
    project     = "helixflow"
    managed_by  = "terraform"
  }
}

module "vpc" {
  source  = "terraform-google-modules/network/google"
  version = "~> 7.0"

  project_id   = var.project_id
  network_name = "helixflow-${var.environment}"
  routing_mode = "GLOBAL"

  subnets = [
    {
      subnet_name   = "private"
      subnet_ip     = var.private_subnet_cidr
      subnet_region = var.region
    },
    {
      subnet_name   = "public"
      subnet_ip     = var.public_subnet_cidr
      subnet_region = var.region
    }
  ]

  secondary_ranges = {
    private = [
      {
        range_name    = "pods"
        ip_cidr_range = "10.1.0.0/16"
      },
      {
        range_name    = "services"
        ip_cidr_range = "10.2.0.0/16"
      }
    ]
  }
}

module "gke" {
  source  = "terraform-google-modules/kubernetes-engine/google"
  version = "~> 29.0"

  project_id = var.project_id
  name       = "helixflow-${var.environment}"
  region     = var.region
  zones      = var.zones

  network    = module.vpc.network_name
  subnetwork = module.vpc.subnets["${var.region}/private"].name

  ip_range_pods     = "pods"
  ip_range_services = "services"

  node_pools = [
    {
      name         = "general"
      machine_type = "e2-standard-4"
      min_count    = 1
      max_count    = 10
      disk_size_gb = 100

      labels = {
        environment = var.environment
        node_group  = "general"
      }
    },
    {
      name         = "gpu"
      machine_type = "n1-standard-4"
      min_count    = 0
      max_count    = 5
      disk_size_gb = 100

      guest_accelerator = {
        type  = "nvidia-tesla-k80"
        count = 1
      }

      labels = {
        environment = var.environment
        node_group  = "gpu"
        gpu         = "enabled"
      }

      taints = [
        {
          key    = "nvidia.com/gpu"
          value  = "present"
          effect = "NoSchedule"
        }
      ]
    }
  ]

  node_pools_labels = {
    all = {
      environment = var.environment
      project     = "helixflow"
    }
  }
}

module "cloud_sql" {
  source  = "GoogleCloudPlatform/sql-db/google"
  version = "~> 16.0"

  project_id = var.project_id
  region     = var.region

  name             = "helixflow-${var.environment}"
  database_version = "POSTGRES_15"
  tier             = "db-f1-micro"

  db_name = "helixflow"
  user_name = "helixflow_admin"

  ip_configuration = {
    ipv4_enabled = false
    private_network = module.vpc.network_self_link
  }

  backup_configuration = {
    enabled    = true
    start_time = "03:00"
  }

  maintenance_window = {
    day  = 7
    hour = 3
  }
}

module "memorystore" {
  source  = "terraform-google-modules/memorystore/google"
  version = "~> 8.0"

  project_id = var.project_id
  region     = var.region

  name = "helixflow-${var.environment}"

  memory_size_gb = 1
  tier           = "STANDARD_HA"

  redis_version = "REDIS_6_X"
}