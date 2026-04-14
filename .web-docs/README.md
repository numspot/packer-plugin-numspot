# Numspot Packer Plugin

The Numspot Packer plugin provides a builder and datasource for creating machine images on the Numspot cloud platform.

## Installation

To install this plugin, add this to your Packer template:

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

## Builders

| Builder | Description |
|---------|-------------|
| [bsu](./components/builder/bsu) | Creates Numspot Images from source images using BSU-backed volumes |

## Data Sources

| Datasource | Description |
|------------|-------------|
| [image](./components/datasource/image) | Resolves a Numspot image by filter before the build starts |

## Authentication

The plugin uses OAuth2 client credentials flow. Configure authentication via:

- `client_id` - OAuth2 client ID (UUID format)
- `client_secret` - OAuth2 client secret
- `space_id` - Numspot space ID (UUID format)

Credentials can be provided in the Packer template or via environment variables:

```bash
export NUMSPOT_CLIENT_ID="<your-client-id>"
export NUMSPOT_CLIENT_SECRET="<your-client-secret>"
export NUMSPOT_SPACE_ID="<your-space-id>"
```

## Prerequisites

Before using this plugin, ensure:

1. **Numspot Account** - OAuth2 credentials from Numspot console
2. **Network Infrastructure**:
   - VPC with Internet Gateway attached
   - Subnet in the VPC
   - Route table with `0.0.0.0/0` → IGW
3. **Permissions** - Service account needs `compute.all` permission

## Quick Start

```hcl
source "numspot-bsu" "example" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = "ami-54a37c4a"
  image_name   = "my-app-${timestamp()}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "outscale"
}

build {
  sources = ["source.numspot-bsu.example"]
}
```

## Links

- [Numspot Documentation](https://docs.numspot.com)
- [GitHub Repository](https://github.com/numspot/packer-plugin-numspot)
- [Issue Tracker](https://github.com/numspot/packer-plugin-numspot/issues)
