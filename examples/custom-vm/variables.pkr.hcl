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

variable "vm_type" {
  type        = string
  default     = "ns-eco7-4c4r"
  description = "VM type to use for the build"
}

variable "image_name" {
  type        = string
  default     = "custom-vm-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  description = "Name for the resulting image"
}

variable "subnet_id" {
  type        = string
  description = "Subnet ID with Internet Gateway and route table setup (strongly recommended to specify explicitly)"
}
