Type: `numspot-bsu`
Artifact BuilderId: `numspot.bsu`

The `numspot-bsu` Packer builder creates Numspot Images backed by BSU (Block Storage Unit) volumes. It launches a VM from a source image, provisions it, and then creates an image from that VM.

## How It Works

1. **Find source image** - Locate the base image by ID or filter
2. **Create temporary resources** - Keypair, security group, public IP
3. **Launch VM** - Start a VM from the source image
4. **Provision** - Run provisioners (shell, file, etc.) on the running VM
5. **Stop VM** - Stop the VM for image creation
6. **Create image** - Create an image from the VM's volumes
7. **Cleanup** - Remove temporary resources (VM, keypair, security group)

## Configuration

### Required

| Option | Type | Description |
|--------|------|-------------|
| `client_id` | string | OAuth2 client ID (UUID). Can use `NUMSPOT_CLIENT_ID` env var. |
| `client_secret` | string | OAuth2 client secret. Can use `NUMSPOT_CLIENT_SECRET` env var. |
| `space_id` | string | Numspot space ID (UUID). Can use `NUMSPOT_SPACE_ID` env var. |
| `source_image` | string | Source image ID, or use `source_image_filter`. |
| `image_name` | string | Name for the resulting image (3-128 chars, alphanumeric/hyphens/underscores). |
| `vm_type` | string | VM type, e.g. `ns-eco7-2c2r`. |

### Optional

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `region` | string | `eu-west-2` | Numspot region. |
| `subnet_id` | string | auto | Subnet ID. Auto-discovered if only one exists. |
| `associate_public_ip_address` | bool | `true` | Associate public IP to VM. |
| `ssh_username` | string | varies | SSH username (depends on source image). |
| `skip_create_image` | bool | `false` | Skip image creation. |
| `disable_stop_vm` | bool | `false` | Don't stop VM before image creation. |
| `force_deregister` | bool | `false` | Deregister existing image with same name. |
| `force_delete_snapshot` | bool | `false` | Delete snapshots when deregistering. |

### Network Filters

Use filters to discover resources dynamically:

```hcl
# Subnet filter
subnet_filter {
  name      = "tag:Environment"
  values    = ["production"]
  most_free = true
}

# VPC filter
net_filter {
  name   = "tag:Name"
  values = ["my-vpc"]
}
```

### Source Image Filter

Instead of `source_image`, use a filter:

```hcl
source_image_filter {
  name        = "name"
  values      = ["ubuntu-22.04-*"]
  owners      = ["numspot"]
  most_recent = true
}
```

### Block Devices

Configure volumes for the build VM:

```hcl
launch_block_device_mappings {
  device_name           = "/dev/sda1"
  volume_size           = 20
  volume_type           = "gp2"
  delete_on_vm_deletion = true
}
```

Configure volumes for the resulting image:

```hcl
image_block_device_mappings {
  device_name = "/dev/sda1"
  volume_size = 20
  volume_type = "gp2"
  snapshot_id = "snap-12345678"
}
```

### Tags

Apply tags to resources:

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

## VM Types

Use Numspot VM types:

- `ns-eco7-2c2r`
- `ns-eco7-4c4r`

**Warning:** Do NOT use Outscale VM types (e.g., `tinav5.c1r1p1`). They are accepted by the API but will fail on Numspot.

## SSH Usernames

Choose the correct username for your source image:

| Image Type | Username |
|------------|----------|
| Rancher/Outscale (Debian) | `outscale` |
| Ubuntu | `ubuntu` |
| Debian | `debian` |
| CentOS | `centos` |

## Example: Basic Build

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
  type = string
}

variable "client_secret" {
  type      = string
  sensitive = true
}

variable "space_id" {
  type = string
}

source "numspot-bsu" "debian" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = "ami-54a37c4a"
  image_name   = "my-app-${timestamp()}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "outscale"

  tags = {
    Name = "my-app"
  }
}

build {
  sources = ["source.numspot-bsu.debian"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y nginx"
    ]
  }
}
```

## Example: With Custom Volumes

```hcl
source "numspot-bsu" "custom" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = "ami-54a37c4a"
  image_name   = "custom-volumes-${timestamp()}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "outscale"

  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 40
    volume_type           = "gp2"
    delete_on_vm_deletion = true
  }

  launch_block_device_mappings {
    device_name = "/dev/sdb"
    volume_size = 100
    volume_type = "gp2"
  }
}
```

## Example: Using Filters

```hcl
source "numspot-bsu" "filtered" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image_filter {
    name        = "name"
    values      = ["ubuntu-22.04-*"]
    most_recent = true
  }

  subnet_filter {
    name      = "tag:Environment"
    values    = ["production"]
    most_free = true
  }

  image_name   = "filtered-${timestamp()}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "ubuntu"
}
```

## Permissions

The service account needs **`compute.all`** permission on the specified space. This grants access to:

- VM lifecycle (create, stop, delete)
- Image lifecycle (create, read, delete)
- Network operations (subnets, security groups, public IPs)
- Tags and keypairs

## Network Requirements

For `associate_public_ip_address = true` (default):

1. **Internet Gateway** attached to VPC
2. **Route table** with `0.0.0.0/0` → IGW
3. **Subnet** in the VPC (auto-discovered or specified)

## Troubleshooting

### Authentication Errors

- Verify `client_id` and `client_secret` are correct
- Check credentials haven't been revoked
- Ensure `client_id` is valid UUID format

### SSH Connection Errors

- Check Internet Gateway is attached to VPC
- Verify route `0.0.0.0/0` → IGW exists
- Use correct SSH username for source image
- Wait for cloud-init to complete (1-2 minutes for Rancher images)

### Permission Errors (403)

- Grant `compute.all` permission to service account

### VM Type Errors

- Use Numspot VM types (`ns-eco7-2c2r`), not Outscale types (`tinav5.c1r1p1`)
