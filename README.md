# Numspot Packer Plugin

A Packer plugin for building machine images on Numspot cloud platform.

## Overview

This plugin enables building Numspot Images from source images using Packer. It uses the Numspot REST API with OAuth2 authentication.

## Quick Links

- [Configuration Reference](./docs/configuration.md)
- [Troubleshooting Guide](./docs/troubleshooting.md)
- [ADR-0001: Design Document](./adr/0001-packer-plugin-numspot.md)
- [API Mapping Reference](./API_MAPPING.md)

## Prerequisites

Before using this plugin, ensure the following:

### 1. Numspot Account & Credentials

- OAuth2 credentials (`client_id`, `client_secret`) from Numspot console
- Space ID where resources will be created

### 2. Network Infrastructure

The following network infrastructure **must be properly configured** before running Packer with `associate_public_ip_address = true` (default):

| Resource | Requirement | Why Required |
|----------|-------------|--------------|
| **VPC** | Must exist with Internet Gateway attached | Internet Gateway is required to associate public IPs to VMs. Without IGW, public IP linking will fail with 409 Conflict error. |
| **Internet Gateway** | Attached to VPC | Enables internet connectivity for VMs with public IPs |
| **Subnet** | At least one subnet in the VPC | VMs are launched in a subnet. Auto-discovered if only one exists, or specify `subnet_id`. |
| **Route Table** | Associated with subnet, route `0.0.0.0/0` → Internet Gateway | Routes outbound traffic from VM to internet via IGW |

> **Important:** If the VPC does not have an Internet Gateway attached, the plugin will create the VM and allocate a public IP, but **cannot link the public IP**. This results in:
> - SSH connection timeout (no public IP reachable)
> - Build hangs waiting for SSH
> - Error: `409 Conflict: Resource conflict` when linking public IP

#### Checking Your VPC Configuration

Verify your network setup before running Packer:

```bash
# 1. Get subnet's VPC ID
curl -H "Authorization: Bearer $TOKEN" \
  "${NUMSPOT_HOST}/compute/spaces/${NUMSPOT_SPACE_ID}/subnets/${SUBNET_ID}" | jq '{vpcId, availabilityZoneName}'

# 2. Verify VPC has Internet Gateway attached
curl -H "Authorization: Bearer $TOKEN" \
  "${NUMSPOT_HOST}/compute/spaces/${NUMSPOT_SPACE_ID}/vpcs/${VPC_ID}" | jq '.internetGatewayId'
# Should return IGW ID (not null)

# 3. Verify route table is associated with subnet
curl -H "Authorization: Bearer $TOKEN" \
  "${NUMSPOT_HOST}/compute/spaces/${NUMSPOT_SPACE_ID}/routeTables" | jq '.items[] | select(.subnetId == "'${SUBNET_ID}'")'

# 4. CRITICAL: Verify route 0.0.0.0/0 exists and points to IGW
curl -H "Authorization: Bearer $TOKEN" \
  "${NUMSPOT_HOST}/compute/spaces/${NUMSPOT_SPACE_ID}/routeTables" | \
  jq '.items[] | select(.vpcId == "'${VPC_ID}'") | .routes[] | select(.destination == "0.0.0.0/0")'
# Should return: {"destination": "0.0.0.0/0", "target": "igw-xxxxx"}
```

#### Quick Setup

If you don't have a VPC with IGW, you can:

1. **Use an existing VPC** that already has IGW attached
2. **Create and attach IGW** to your current VPC:
   ```bash
   # 1. Create Internet Gateway
   # 2. Attach IGW to VPC
   # 3. Create or identify a route table
   # 4. Associate route table with subnet
   # 5. CRITICAL: Add route 0.0.0.0/0 → IGW in route table
   #    Without this route, VMs cannot reach internet even with IGW attached!
   ```

> **Common Mistake:** Creating and attaching Internet Gateway is necessary but **not sufficient**. You MUST also:
> 1. Associate the route table with your subnet
> 2. Add a route in that table: `0.0.0.0/0` → `igw-xxxxx`
> 
> Without the route, builds will timeout waiting for SSH even though everything else is configured.

> **Note:** The plugin auto-discovers a subnet if only one exists. If multiple subnets exist, specify `subnet_id` or use `subnet_filter`.

### 3. Required Permissions

The service account needs **`compute.all`** permission on the specified space.

This single permission grants access to all operations needed:
- VM lifecycle (create, stop, delete)
- Image lifecycle (create, read, delete)
- Network (subnets, security groups, public IPs)
- Tags and keypairs

## Installation

### From Source

```bash
git clone <repository-url>
cd numspot-plugin-packer
go build -o packer-plugin-numspot
packer plugins install github.com/numspot/numspot ./packer-plugin-numspot
```

### Using `packer init`

```hcl
packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}
```

Then run:

```bash
packer init .
```

## Configuration

### Required Parameters

| Parameter | Environment Variable | Description | Example |
|-----------|---------------------|-------------|---------|
| `client_id` | `NUMSPOT_CLIENT_ID` | OAuth2 client ID (UUID) | `<your-client-id>` |
| `client_secret` | `NUMSPOT_CLIENT_SECRET` | OAuth2 client secret | (sensitive) |
| `space_id` | `NUMSPOT_SPACE_ID` | Numspot space ID (UUID) | `<your-space-id>` |

> **Note:** The API endpoint is automatically constructed from the region. Default: `https://api.{region}.numspot.com`.

### Optional Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `region` | `eu-west-2` | Numspot region |
| `skip_create_image` | `false` | Skip image creation |
| `disable_stop_vm` | `false` | Don't stop VM before image creation |
| `associate_public_ip_address` | `true` | Associate public IP to VM |

For all parameters, see the [Configuration Reference](./docs/configuration.md).

## Example Usage

```hcl
packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

variable "client_secret" {
  type        = string
  sensitive   = true
  description = "Numspot OAuth2 client secret"
}

source "numspot-bsu" "example" {
  # Authentication
  client_id     = "<your-client-id>"
  client_secret = var.client_secret
  
  # Space (region defaults to eu-west-2)
  space_id = "<your-space-id>"
  
  # Source image (Debian 13 Trixie)
  source_image = "ami-54a37c4a"
  
  # Resulting image
  image_name = "my-app-${timestamp()}"
  
  # VM configuration
  vm_type = "ns-eco7-2c2r"
  
  # Network (optional - auto-discovered if not specified)
  subnet_id = "<your-subnet-id>"
  
  # SSH configuration
  ssh_username = "outscale"  # For Rancher/Outscale images
  
  # Tags
  tags = {
    Name        = "my-app"
    Environment = "production"
  }
}

build {
  sources = ["source.numspot-bsu.example"]
  
  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y nginx"
    ]
  }
}
```

Or use environment variables:

```bash
export NUMSPOT_CLIENT_ID="<your-client-id>"
export NUMSPOT_CLIENT_SECRET="<your-client-secret>"
export NUMSPOT_SPACE_ID="<your-space-id>"
# NUMSPOT_REGION is optional (defaults to eu-west-2)

packer build template.pkr.hcl
```

## Datasource: `numspot-bsu` image

The `image` datasource resolves a Numspot image by filter **before** the build starts.
It is useful when you don't want to hardcode an image ID in your template.

### Usage

```hcl
data "numspot-image" "base" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  filters = {
    name = "Ubuntu-22.04-latest"
  }
  most_recent = true
}

source "numspot-bsu" "example" {
  # ...
  source_image = data.numspot-image.base.id
}
```

### Filter keys

| Key | Description |
|---|---|
| `name` | Image name (exact match) |
| `id` | Image ID |
| `state` | Image state (`available`, `pending`, `failed`) |
| `architecture` | Architecture (e.g. `x86_64`) |
| `description` | Image description |
| `root_device_type` | Root device type |
| `root_device_name` | Root device name |

### Output attributes

| Attribute | Description |
|---|---|
| `id` | Image ID (e.g. `ami-bf56f9be`) |
| `name` | Image name |
| `description` | Image description |
| `creation_date` | Creation date (ISO 8601) |
| `state` | Image state |
| `architecture` | Architecture |
| `root_device_name` | Root device name |
| `root_device_type` | Root device type |
| `tags` | Map of tags |

> **Note:** Unlike the Outscale equivalent, `owners` is optional — your space already scopes image visibility.

See the [image-filter example](./examples/image-filter/) for a full working template.

## Builder: `numspot-bsu`

The BSU (Block Storage Unit) builder creates images from existing source images.

### Workflow

1. **Find source image** by ID or filter
2. **Create temporary keypair** for SSH access
3. **Launch VM** from source image
4. **Provision VM** using configured provisioners
5. **Stop VM** (unless `disable_stop_vm = true`)
6. **Create image** from VM
7. **Clean up** VM, keypair, security group

### Required Configuration

- `source_image` - ID of the source image (or use `source_image_filter`)
- `image_name` - Name for the resulting image
- `vm_type` - VM type to use during build
- `subnet_id` - **Strongly recommended**: Subnet ID with proper network setup

> **Important:** While `subnet_id` is technically optional (auto-discovery works with single subnet), explicit configuration is **strongly recommended** to ensure:
> - Subnet has Internet Gateway attached to its VPC
> - Route table is properly associated with subnet
> - Route `0.0.0.0/0` points to IGW
> - Build configuration is explicit and reproducible
>
> Auto-discovery can fail silently if the discovered subnet lacks proper network configuration, resulting in SSH timeouts or public IP linking failures.

### Network Requirements

For `associate_public_ip_address = true` (default):

1. **Internet Gateway** attached to VPC
2. **Route table** associated with subnet, route `0.0.0.0/0` → IGW
3. **Security Group** created automatically (allows SSH port 22)
4. **Subnet** - specify explicitly for reliable builds:

```hcl
subnet_id = "<your-subnet-id>"  # Subnet with IGW + route table setup
```

### VM Types

Use Numspot VM types (e.g., `ns-eco7-2c2r`, `ns-eco7-4c4r`).

> **Warning:** Do NOT use Outscale VM types (e.g., `tinav5.c1r1p1`).

For the complete list of available VM types, see: [Numspot VM Types Documentation](https://docs.numspot.com/docs/compute/vms/types)

### SSH Username

The SSH username depends on the source image:

| Image Type | Username |
|------------|----------|
| Ubuntu images | `ubuntu` |
| Debian images | `debian` |
| CentOS images | `centos` |

Check the source image documentation for the correct username.

## Authentication

The plugin uses OAuth2 client credentials flow:

1. Client sends `client_id` and `client_secret` to token endpoint
2. Receives `access_token` valid for 1 hour
3. Token is automatically refreshed when needed

Token endpoint: `POST https://api.{region}.numspot.com/iam/token`

## SDK Architecture

The plugin uses a generated SDK from the Numspot OpenAPI spec:

```
numspot/
├── client.go       # OAuth2 client with token management
├── helpers.go      # Wait helpers for async operations
├── api.gen.go      # Generated REST API client
└── types.gen.go    # Generated type definitions
```

## Development

### Prerequisites

- Go 1.22+
- Packer SDK v0.6+
- oapi-codegen v2 (for SDK regeneration)

### Building

```bash
make build
```

### Testing

```bash
# Unit tests
make test

# Integration tests (requires credentials)
source .env
make test-integration
```

### Linting

```bash
make lint
```

### Regenerating SDK

```bash
oapi-codegen -generate types -package numspot \
  ~/Numspot/eclused-compute/eclused-compute/api/bundled-eclused-compute.yaml > numspot/types.gen.go

oapi-codegen -generate client -package numspot \
  ~/Numspot/eclused-compute/eclused-compute/api/bundled-eclused-compute.yaml > numspot/api.gen.go
```

## Troubleshooting

See the [Troubleshooting Guide](./docs/troubleshooting.md) for common issues and solutions.

## License

[License](./LICENSE)

## References

- [Outscale Packer Plugin](https://github.com/outscale/packer-plugin-outscale) (forked)
- [Numspot Terraform Provider](https://github.com/numspot/terraform-provider-numspot)
- [Packer Plugin SDK](https://github.com/hashicorp/packer-plugin-sdk)
- [oapi-codegen](https://github.com/deepmap/oapi-codegen)
