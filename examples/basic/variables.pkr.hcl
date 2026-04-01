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

variable "subnet_id" {
  type        = string
  default     = ""
  description = "Subnet ID (leave empty for auto-discovery)"
}
