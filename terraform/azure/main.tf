terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }

  backend "azurerm" {
    resource_group_name  = "helixflow-terraform"
    storage_account_name = "helixflowtfstate"
    container_name       = "tfstate"
    key                  = "azure-infrastructure.tfstate"
  }
}

provider "azurerm" {
  features {}

  subscription_id = var.subscription_id
  tenant_id       = var.tenant_id

  default_tags = {
    Environment = var.environment
    Project     = "helixflow"
    ManagedBy   = "terraform"
  }
}

module "resource_group" {
  source = "../modules/resource-group"

  name     = "helixflow-${var.environment}"
  location = var.location

  tags = local.common_tags
}

module "vnet" {
  source = "../modules/vnet"

  name                = "helixflow-${var.environment}"
  resource_group_name = module.resource_group.name
  location            = module.resource_group.location
  address_space       = [var.vnet_cidr]

  subnets = {
    private = {
      name             = "private"
      address_prefixes = var.private_subnets
    }
    public = {
      name             = "public"
      address_prefixes = var.public_subnets
    }
  }

  tags = local.common_tags
}

module "aks" {
  source  = "Azure/aks/azurerm"
  version = "~> 7.0"

  resource_group_name = module.resource_group.name
  location            = module.resource_group.location

  prefix          = "helixflow"
  cluster_name    = "helixflow-${var.environment}"
  kubernetes_version = "1.28.0"

  vnet_subnet_id = module.vnet.subnets["private"].id

  default_node_pool = {
    name                = "general"
    node_count          = 2
    vm_size             = "Standard_D4s_v3"
    vnet_subnet_id      = module.vnet.subnets["private"].id
    enable_auto_scaling = true
    min_count           = 1
    max_count           = 10
  }

  node_pools = {
    gpu = {
      name                = "gpu"
      vm_size             = "Standard_NC6s_v3"
      node_count          = 1
      vnet_subnet_id      = module.vnet.subnets["private"].id
      enable_auto_scaling = true
      min_count           = 0
      max_count           = 5

      node_labels = {
        "accelerator" = "nvidia-tesla-k80"
      }

      node_taints = [
        "nvidia.com/gpu=present:NoSchedule"
      ]
    }
  }

  network_profile = {
    network_plugin    = "azure"
    network_policy    = "azure"
    load_balancer_sku = "standard"
  }

  tags = local.common_tags
}

module "postgresql" {
  source  = "Azure/postgresql/azurerm"
  version = "~> 3.0"

  resource_group_name = module.resource_group.name
  location            = module.resource_group.location

  server_name        = "helixflow-${var.environment}"
  sku_name           = "GP_Standard_D4s_v3"
  storage_mb         = 102400

  postgresql_version = "15"

  administrator_login    = "helixflow_admin"
  administrator_password = var.postgres_password

  database_names = ["helixflow"]

  firewall_rules = [
    {
      name             = "AllowAzureServices"
      start_ip_address = "0.0.0.0"
      end_ip_address   = "0.0.0.0"
    }
  ]

  tags = local.common_tags
}

module "redis" {
  source  = "Azure/redis-cache/azurerm"
  version = "~> 3.0"

  resource_group_name = module.resource_group.name
  location            = module.resource_group.location

  name     = "helixflow-${var.environment}"
  capacity = 1
  family   = "C"
  sku_name = "Standard"

  redis_version = "6"

  tags = local.common_tags
}

locals {
  common_tags = {
    Environment = var.environment
    Project     = "helixflow"
    ManagedBy   = "terraform"
  }
}