packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

# Resolve the source image dynamically before the build starts.
# This avoids hardcoding an image ID and always picks the latest matching image.
data "numspot-bsu" "image" "base" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  filters = {
    name = var.source_image_name
  }
  most_recent = true
}

source "numspot-bsu" "image-filter" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  # Image ID resolved by the datasource above
  source_image = data.numspot-bsu.image.base.id

  image_name   = "image-filter-example-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "outscale"

  subnet_id                   = var.subnet_id
  associate_public_ip_address = true
  ssh_interface               = "public_ip"

  tags = {
    Name        = "image-filter-example"
    SourceImage = data.numspot-bsu.image.base.name
    Environment = "dev"
    ManagedBy   = "packer"
  }
}

build {
  sources = ["source.numspot-bsu.image-filter"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y nginx",
      "sudo systemctl enable nginx",
    ]
  }
}
