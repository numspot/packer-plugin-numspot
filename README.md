# Numspot Packer Plugin

A Packer plugin for building machine images on Numspot cloud platform.

## Overview

This plugin enables building Numspot Images from source images using Packer. It uses the Numspot REST API with OAuth2 authentication.

## Quick Links

- [Configuration Reference](./docs/configuration.md)
- [Troubleshooting Guide](./docs/troubleshooting.md)
- [ADR-0001: Design Document](./adr/0001-packer-plugin-numspot.md)
- [API Mapping Reference](./API_MAPPING.md)

## Project Status

| Phase | Status |
|-------|--------|
| Phase 1: SDK Generation | ✅ Complete |
| Phase 2: Plugin Adaptation | ✅ Complete |
| Phase 3: HCL Generation | ✅ Complete |
| Phase 4: Testing & Documentation | ✅ Complete |

## Prerequisites

Before using this plugin, ensure the following:

### 1. Numspot Account & Credentials

- OAuth2 credentials (`client_id`, `client_secret`) from Numspot console
- Space ID where resources will be created

### 2. Network Infrastructure

The following must exist before running Packer:

| Resource | Requirement |
|----------|-------------|
| **VPC** | Must exist with Internet Gateway attached |
| **Subnet** | At least one subnet in the VPC |
| **Route Table** | Route `0.0.0.0/0` → Internet Gateway (for public IP access) |

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

### Network Requirements

For `associate_public_ip_address = true` (default):

1. **Internet Gateway** attached to VPC
2. **Route table** with `0.0.0.0/0` → IGW
3. **Security Group** created automatically (allows SSH port 22)
4. **Subnet** - auto-discovered if not specified, or set explicitly:

```hcl
subnet_id = "<your-subnet-id>"
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

[License information]

## References

- [Outscale Packer Plugin](https://github.com/outscale/packer-plugin-outscale) (original implementation)
- [Numspot Terraform Provider](https://github.com/numspot/terraform-provider-numspot)
- [Packer Plugin SDK](https://github.com/hashicorp/packer-plugin-sdk)
- [oapi-codegen](https://github.com/deepmap/oapi-codegen)
