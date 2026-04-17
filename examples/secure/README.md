# Security Hardened Example

Creates a security-hardened image with firewall, automatic updates, and SSH hardening.

## Usage

```bash
# Set required variables
export PKR_VAR_client_id="<your-client-id>"
export PKR_VAR_client_secret="<your-client-secret>"
export PKR_VAR_space_id="<your-space-id>"
export PKR_VAR_subnet_id="<your-subnet-id>"

# Use either datasource with image name or directly imageId
export PKR_VAR_source_image_name="<your-image-name>"

# Optionally restrict SSH access to your IP range
export PKR_VAR_allowed_ssh_cidr="203.0.113.0/24"

# Build
packer build .
```

## Security Features

### Network Security
- **Restricted SSH CIDR** - Limits SSH access to specified IP range (default: `10.0.0.0/8`)
- Only internal network access during build

### Firewall (UFW)
- Default deny incoming traffic
- Default allow outgoing traffic
- Only SSH (port 22) allowed

### SSH Hardening
- Root login disabled
- Password authentication disabled
- Public key authentication only

### Automatic Updates
- `unattended-upgrades` enabled
- Automatic security patches

### Intrusion Prevention
- `fail2ban` installed (default config)

### Cleanup
- Package cache cleared
- Temporary files removed
- Logs cleared
- Shell history cleared

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `source_image_name` | `debian-11` | Source image name pattern to filter |
| `subnet_id` | (required) | Subnet ID with IGW and route table setup |

> **Note:** The image datasource dynamically resolves the latest matching image, avoiding hardcoded AMI IDs.

## Files

| File | Description |
|------|-------------|
| `main.pkr.hcl` | Main Packer template with security hardening |
| `variables.pkr.hcl` | Variable definitions |
