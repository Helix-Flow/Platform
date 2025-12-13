variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-east1"
}

variable "zones" {
  description = "List of GCP zones"
  type        = list(string)
  default     = ["us-east1-b", "us-east1-c", "us-east1-d"]
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "development"
}

variable "private_subnet_cidr" {
  description = "CIDR for private subnet"
  type        = string
  default     = "10.0.1.0/24"
}

variable "public_subnet_cidr" {
  description = "CIDR for public subnet"
  type        = string
  default     = "10.0.101.0/24"
}