packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

source "numspot-bsu" "custom" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = "ami-54a37c4a"
  image_name   = var.image_name
  vm_type      = var.vm_type
  ssh_username = "outscale"

  # Required: Subnet with Internet Gateway and route table setup
  subnet_id                   = var.subnet_id
  associate_public_ip_address = true
  ssh_interface               = "public_ip"

  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 50
    volume_type           = "gp2"
    delete_on_vm_deletion = true
  }

  tags = {
    Name        = "custom-vm-example"
    VMType      = var.vm_type
    Environment = "dev"
    ManagedBy   = "packer"
  }
}

build {
  sources = ["source.numspot-bsu.custom"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y docker.io docker-compose",
      "sudo usermod -aG docker outscale",
      "sudo systemctl enable docker"
    ]
  }
}
