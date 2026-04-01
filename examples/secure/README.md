# Security Hardened Example

Creates a security-hardened image with firewall, automatic updates, and SSH hardening.

## Usage

```bash
# Set required variables
export PKR_VAR_client_id="<your-client-id>"
export PKR_VAR_client_secret="<your-client-secret>"
export PKR_VAR_space_id="<your-space-id>"

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
| `allowed_ssh_cidr` | `10.0.0.0/8` | CIDR block for SSH access during build |

> **Important:** Set `allowed_ssh_cidr` to your specific IP range for production builds.

## Files

| File | Description |
|------|-------------|
| `main.pkr.hcl` | Main Packer template with security hardening |
| `variables.pkr.hcl` | Variable definitions |
