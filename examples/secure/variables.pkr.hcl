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

variable "allowed_ssh_cidr" {
  type        = string
  default     = "10.0.0.0/8"
  description = "CIDR block allowed for SSH access (restrict to your IP range)"
}

variable "subnet_id" {
  type        = string
  default     = ""
  description = "Subnet ID (leave empty for auto-discovery)"
}
