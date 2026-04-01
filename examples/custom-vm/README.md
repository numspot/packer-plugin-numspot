# Custom VM Example

Demonstrates how to use custom VM types and volume configurations.

## Usage

```bash
# Set required variables
export PKR_VAR_client_id="<your-client-id>"
export PKR_VAR_client_secret="<your-client-secret>"
export PKR_VAR_space_id="<your-space-id>"

# Build with defaults (ns-eco7-4c4r)
packer build .

# Or override VM type
export PKR_VAR_vm_type="ns-eco7-2c2r"
packer build .
```

## Available VM Types

| Type | vCPUs | RAM |
|------|-------|-----|
| `ns-eco7-2c2r` | 2 | 2 GB |
| `ns-eco7-4c4r` | 4 | 4 GB |

> **Note:** For the complete list, see [Numspot VM Types Documentation](https://docs.numspot.com/docs/compute/vms/types)

## What It Does

1. Uses a larger VM type (`ns-eco7-4c4r` by default)
2. Configures a 50GB gp2 volume
3. Installs Docker and Docker Compose
4. Creates an image with Docker pre-installed

## Files

| File | Description |
|------|-------------|
| `main.pkr.hcl` | Main Packer template |
| `variables.pkr.hcl` | Variable definitions including `vm_type` |
