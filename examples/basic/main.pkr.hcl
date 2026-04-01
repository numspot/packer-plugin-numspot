packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

source "numspot-bsu" "basic" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = "ami-54a37c4a"
  image_name   = "basic-example-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "outscale"

  subnet_id                   = var.subnet_id != "" ? var.subnet_id : null
  associate_public_ip_address = true
  ssh_interface               = "public_ip"

  tags = {
    Name        = "basic-example"
    Environment = "dev"
    ManagedBy   = "packer"
  }
}

build {
  sources = ["source.numspot-bsu.basic"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y nginx",
      "sudo systemctl enable nginx",
      "echo 'Basic example completed'",
    ]
  }
}
