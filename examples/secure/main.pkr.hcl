packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

source "numspot-bsu" "secure" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = "ami-54a37c4a"
  image_name   = "secure-example-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "outscale"

  subnet_id                            = var.subnet_id != "" ? var.subnet_id : null
  associate_public_ip_address          = true
  ssh_interface                        = "public_ip"

  tags = {
    Name        = "secure-example"
    Environment = "production"
    ManagedBy   = "packer"
    Security    = "hardened"
  }

  run_tags = {
    Name      = "packer-build-secure"
    Temporary = "true"
  }
}

build {
  sources = ["source.numspot-bsu.secure"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get upgrade -y",
      "sudo apt-get install -y ufw fail2ban unattended-upgrades",
      "echo 'unattended-upgrades unattended-upgrades/enable_auto_updates boolean true' | sudo debconf-set-selections",
      "sudo DEBIAN_FRONTEND=noninteractive dpkg-reconfigure -f noninteractive unattended-upgrades",
    ]
  }

  provisioner "shell" {
    inline = [
      "sudo ufw default deny incoming",
      "sudo ufw default allow outgoing",
      "sudo ufw allow ssh",
      "sudo ufw --force enable",
    ]
  }

  provisioner "shell" {
    inline = [
      "sudo sed -i 's/#PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config",
      "sudo sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config",
      "sudo sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
      "sudo systemctl restart sshd",
    ]
  }

  provisioner "shell" {
    inline = [
      "sudo apt-get autoremove -y",
      "sudo apt-get clean",
      "sudo rm -rf /var/lib/apt/lists/*",
      "sudo rm -rf /tmp/*",
      "sudo rm -rf /var/tmp/*",
      "cat /dev/null | sudo tee /var/log/syslog",
      "sudo history -c",
    ]
  }
}
