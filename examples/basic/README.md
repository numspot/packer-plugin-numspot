# Basic Example

A minimal example that builds a Numspot image from a Rancher Debian base image with nginx installed.

## Usage

```bash
# Set required variables
export PKR_VAR_client_id="<your-client-id>"
export PKR_VAR_client_secret="<your-client-secret>"
export PKR_VAR_space_id="<your-space-id>"

# Build
packer build .
```

## What It Does

1. Uses Rancher Debian 10 image (`ami-52b3214f`)
2. Launches a `ns-eco7-2c2r` VM
3. Installs nginx via shell provisioner
4. Creates an image named `basic-example-YYYY-MM-DD-hhmm`
5. Applies tags for organization

## Files

| File | Description |
|------|-------------|
| `main.pkr.hcl` | Main Packer template |
| `variables.pkr.hcl` | Variable definitions |
