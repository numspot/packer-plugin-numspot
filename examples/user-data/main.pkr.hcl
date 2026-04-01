packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

locals {
  user_data = templatefile("${path.root}/scripts/user-data.sh", {
    app_name    = var.application_name
    app_version = var.application_version
  })
}

source "numspot-bsu" "userdata" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = "ami-54a37c4a"
  image_name   = "userdata-example-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "outscale"

  subnet_id                   = var.subnet_id != "" ? var.subnet_id : null
  associate_public_ip_address = true
  ssh_interface               = "public_ip"
  user_data_file              = "${path.root}/scripts/cloud-init.yaml"

  tags = {
    Name        = "userdata-example"
    AppName     = var.application_name
    AppVersion  = var.application_version
    Environment = "dev"
    ManagedBy   = "packer"
  }
}

build {
  sources = ["source.numspot-bsu.userdata"]

  provisioner "shell" {
    inline = [
      "echo 'Checking cloud-init status...'",
      "cloud-init status --wait || true",
      "echo 'Application should be pre-installed via user-data'",
    ]
  }

  provisioner "shell" {
    inline = [
      "sudo systemctl status nginx || echo 'nginx not running'",
      "ls -la /opt/webapp || echo 'webapp directory not found'",
    ]
  }
}
