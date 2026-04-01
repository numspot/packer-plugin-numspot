# Configuration Reference

This document describes all configuration options for the Numspot Packer plugin.

## Table of Contents

- [Required Parameters](#required-parameters)
- [Network Configuration](#network-configuration)
- [Source Image Configuration](#source-image-configuration)
- [Image Configuration](#image-configuration)
- [VM Configuration](#vm-configuration)
- [SSH Configuration](#ssh-configuration)
- [Block Devices](#block-devices)
- [Tags](#tags)

---

## Required Parameters

These parameters must be provided either in the Packer template or via environment variables.

| Parameter | Environment Variable | Type | Description |
|-----------|---------------------|------|-------------|
| `client_id` | `NUMSPOT_CLIENT_ID` | string | OAuth2 client ID (UUID format) |
| `client_secret` | `NUMSPOT_CLIENT_SECRET` | string | OAuth2 client secret |
| `space_id` | `NUMSPOT_SPACE_ID` | string | Numspot space ID (UUID format) |

### Optional Connection Parameters

| Parameter | Environment Variable | Type | Default | Description |
|-----------|---------------------|------|---------|-------------|
| `region` | `NUMSPOT_REGION` | string | `eu-west-2` | Numspot region |

> **Note:** The API endpoint is automatically constructed as `https://api.{region}.numspot.com`.

### Example

```hcl
source "numspot-bsu" "example" {
  client_id     = "<your-client-id>"
  client_secret = var.client_secret
  space_id      = "<your-space-id>"
  region        = "eu-west-2"  # Optional, defaults to eu-west-2
}
```

---

## Network Configuration

Configure how the VM connects to the network.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `subnet_id` | string | auto | Subnet ID to launch VM in. If not specified, plugin auto-discovers subnet. |
| `net_id` | string | auto | VPC/Net ID to launch VM in. |
| `associate_public_ip_address` | bool | `true` | Associate a public IP address to the VM. |
| `security_group_ids` | []string | auto | List of security group IDs to attach. |
| `security_group_id` | string | - | (Deprecated) Single security group ID. Use `security_group_ids` instead. |
| `temporary_security_group_source_cidr` | string | `0.0.0.0/0` | CIDR block for temporary security group rules. |

### Filter Options

Instead of specifying IDs directly, you can use filters to discover resources:

#### Subnet Filter (`subnet_filter`)

```hcl
subnet_filter {
  name    = "tag:Environment"
  values  = ["production"]
  most_free = true    # Select subnet with most free IPs
  random    = false   # Random selection from matching subnets
}
```

#### VPC Filter (`net_filter`)

```hcl
net_filter {
  name   = "tag:Name"
  values = ["my-vpc"]
}
```

#### Security Group Filter (`security_group_filter`)

```hcl
security_group_filter {
  name   = "tag:Name"
  values = ["packer-sg"]
}
```

---

## Source Image Configuration

Define the source image to build from.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `source_image` | string | yes* | Source image ID (e.g., `ami-52b3214f`). |
| `source_image_filter` | object | yes* | Filter to discover source image. |

*One of `source_image` or `source_image_filter` is required.

### Direct Image ID

```hcl
source_image = "ami-52b3214f"
```

### Image Filter

```hcl
source_image_filter {
  name        = "name"
  values      = ["ubuntu-22.04-*"]
  owners      = ["numspot"]
  most_recent = true
}
```

| Filter Parameter | Type | Description |
|-----------------|------|-------------|
| `name` | string | Filter name (e.g., `name`, `tag:Environment`) |
| `values` | []string | Filter values |
| `owners` | []string | Account aliases/owners to filter by |
| `most_recent` | bool | Select most recent image when multiple match |

---

## Image Configuration

Configure the resulting image.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `image_name` | string | required | Name of the resulting image (3-128 chars, alphanumeric/hyphens/underscores). |
| `image_description` | string | - | Description of the resulting image. |
| `image_account_ids` | []string | - | Account IDs with launch permission. |
| `image_groups` | []string | - | Groups with launch permission. |
| `image_regions` | []string | - | Regions to copy the image to. |
| `image_boot_modes` | []string | - | Boot modes: `legacy`, `uefi`. |
| `product_codes` | []string | - | Product codes to associate. |
| `global_permission` | bool | `false` | Make image public. |
| `force_deregister` | bool | `false` | Force deregister existing image with same name. |
| `force_delete_snapshot` | bool | `false` | Delete snapshots when deregistering. |
| `root_device_name` | string | - | Root device name (e.g., `/dev/sda1`). |

### Example

```hcl
image_name        = "my-app-${timestamp()}"
image_description = "My custom application image"

tags = {
  Environment = "production"
  BuiltBy     = "packer"
}

product_codes = ["product-code-123"]
```

---

## VM Configuration

Configure the VM used during the build.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `vm_type` | string | required | VM type (e.g., `ns-eco7-2c2r`). |
| `availability_zone` | string | auto | Availability zone name. |
| `shutdown_behavior` | string | `stop` | VM shutdown behavior: `stop` or `terminate`. |
| `boot_mode` | string | - | VM boot mode: `legacy` or `uefi`. |
| `user_data` | string | - | User data (base64 encoded). |
| `user_data_file` | string | - | Path to user data file. |
| `bsu_optimized` | bool | `false` | Enable BSU optimization. |
| `iam_vm_profile` | string | - | IAM VM profile name. |
| `enable_t2_unlimited` | bool | `false` | Enable T2 Unlimited (T2 instances only). |
| `block_duration_minutes` | int | - | Block duration for spot instances (multiple of 60). |

### Valid VM Types

Numspot uses its own VM type naming:

- `ns-eco7-2c2r`
- `ns-eco7-4c4r`

> **Warning:** Do NOT use Outscale VM types (e.g., `tinav5.c1r1p1`). They are accepted by API but will fail on Numspot.

### User Data

```hcl
# Inline (base64 encoded)
user_data = "IyEvYmluL2Jhc2gKZWNobyAiSGVsbG8gV29ybGQi"

# Or from file
user_data_file = "./scripts/init.sh"
```

---

## SSH Configuration

Configure SSH access to the VM.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `ssh_username` | string | varies | SSH username (depends on source image). |
| `ssh_password` | string | - | SSH password. |
| `ssh_private_key_file` | string | - | Path to SSH private key file. |
| `ssh_keypair_name` | string | - | Existing keypair name. |
| `ssh_interface` | string | `public_ip` | SSH interface: `public_ip`, `private_ip`, `public_dns`, `private_dns`. |

### SSH Username by Image

| Image Type | Username |
|------------|----------|
| Rancher/Outscale (Debian) | `outscale` |
| Ubuntu | `ubuntu` |
| Debian | `debian` |
| CentOS | `centos` |

### Example

```hcl
ssh_username        = "outscale"
ssh_private_key_file = "~/.ssh/id_rsa"
```

### Temporary Keypair

If no keypair is specified, the plugin creates a temporary keypair:

```hcl
# Plugin auto-generates: pk-{timestamp}
# Private key is saved to: ~/.packer.d/tmp/{keypair-name}
```

---

## Block Devices

Configure block device mappings.

### Launch Block Devices (`launch_block_device_mappings`)

Configure volumes for the build VM.

```hcl
launch_block_device_mappings {
  device_name          = "/dev/sda1"
  volume_size          = 20
  volume_type          = "gp2"
  delete_on_vm_deletion = true
}
```

### Image Block Devices (`image_block_device_mappings`)

Configure volumes for the resulting image.

```hcl
image_block_device_mappings {
  device_name   = "/dev/sda1"
  volume_size   = 20
  volume_type   = "gp2"
  snapshot_id   = "snap-12345678"
}
```

### Block Device Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `device_name` | string | required | Device name (e.g., `/dev/sda1`). |
| `volume_size` | int | - | Volume size in GiB. |
| `volume_type` | string | `standard` | Volume type: `standard`, `gp2`, `io1`. |
| `iops` | int | - | IOPS for `io1` volumes. |
| `snapshot_id` | string | - | Snapshot ID to create volume from. |
| `delete_on_vm_deletion` | bool | `true` | Delete volume when VM is terminated. |
| `no_device` | bool | `false` | Suppress the device mapping. |

---

## Tags

Apply tags to resources created during the build.

| Parameter | Description |
|-----------|-------------|
| `tags` | Tags applied to the resulting image. |
| `run_tags` | Tags applied to the VM during build. |
| `run_volume_tags` | Tags applied to volumes during build. |
| `snapshot_tags` | Tags applied to snapshots created. |

### Example

```hcl
tags = {
  Name        = "my-app"
  Environment = "production"
}

run_tags = {
  Purpose = "packer-build"
}

run_volume_tags = {
  Type = "build-volume"
}
```

---

## Complete Example

```hcl
packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

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

source "numspot-bsu" "ubuntu" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id
  
  source_image = "ami-54a37c4a"
  
  image_name        = "my-app-${timestamp()}"
  image_description = "Custom application image"
  
  vm_type = "ns-eco7-2c2r"
  
  associate_public_ip_address  = true
  
  ssh_username = "outscale"
  
  # Tags
  tags = {
    Name        = "my-app"
    Environment = "production"
  }
  
  # Block devices
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 20
    volume_type = "gp2"
  }
}

build {
  sources = ["source.numspot-bsu.ubuntu"]
  
  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y nginx"
    ]
  }
}
```

---

## See Also

- [Troubleshooting Guide](./troubleshooting.md)
- [Prerequisites](../README.md#prerequisites)
- [Example Templates](../examples/)
