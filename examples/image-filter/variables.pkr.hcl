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
  description = "Subnet ID with Internet Gateway and route table setup"
}

variable "source_image_name" {
  type        = string
  description = "Exact name of the source image to look up via the datasource"
}
