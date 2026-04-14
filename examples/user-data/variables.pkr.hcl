variable "client_id" {
  type        = string
  description = "Numspot OAuth2 client ID"
}

variable "client_secret" {
  type        = string
  sensitive   = true
  description = "Numspot OAuth2 client secret"
}

variable "space_id" {
  type        = string
  description = "Numspot space ID"
}

variable "application_name" {
  type        = string
  default     = "webapp"
  description = "Application name for deployment"
}

variable "application_version" {
  type        = string
  default     = "1.0.0"
  description = "Application version"
}

variable "subnet_id" {
  type        = string
  description = "Subnet ID with Internet Gateway and route table setup (strongly recommended to specify explicitly)"
}
