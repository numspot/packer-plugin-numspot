packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

data "numspot-image" "base" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  filters = {
    name = var.source_image_name
  }
  most_recent = true
}

source "numspot-bsu" "secure" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = data.numspot-image.base.id
  image_name   = "demo-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "outscale"

  subnet_id                   = var.subnet_id
  associate_public_ip_address = true
  ssh_interface               = "public_ip"

  tags = {
    Name        = "packer-demo"
    SourceImage = data.numspot-image.base.name
    ManagedBy   = "packer"
  }
}

build {
  sources = ["source.numspot-bsu.secure"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y nginx fail2ban",
      "echo 'Demo image built with Packer!' | sudo tee /etc/motd",
    ]
  }
}
