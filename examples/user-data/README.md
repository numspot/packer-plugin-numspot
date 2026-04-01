# User Data Example

Demonstrates how to use `user_data_file` to configure a VM during boot using cloud-init.

## Usage

```bash
# Set required variables
export PKR_VAR_client_id="<your-client-id>"
export PKR_VAR_client_secret="<your-client-secret>"
export PKR_VAR_space_id="<your-space-id>"

# Optionally customize application
export PKR_VAR_application_name="myapp"
export PKR_VAR_application_version="2.0.0"

# Build
packer build .
```

## What It Does

1. Passes `cloud-init.yaml` as user-data to the VM
2. Cloud-init runs during first boot:
   - Updates packages
   - Installs nginx
   - Creates application config files
   - Starts nginx service
3. Packer connects and verifies the setup

## User Data Formats

### cloud-init (YAML)

The recommended format for cloud configuration:

```yaml
#cloud-config

packages:
  - nginx
  - curl

write_files:
  - path: /opt/app/config.json
    content: |
      {"name": "myapp"}

runcmd:
  - systemctl start nginx
```

### Shell Script

You can also use a shell script:

```hcl
user_data_file = "${path.root}/scripts/init.sh"
```

The script will be executed during first boot.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `application_name` | `webapp` | Application name |
| `application_version` | `1.0.0` | Application version |

## Files

| File | Description |
|------|-------------|
| `main.pkr.hcl` | Main Packer template |
| `variables.pkr.hcl` | Variable definitions |
| `scripts/cloud-init.yaml` | Cloud-init configuration |
| `scripts/user-data.sh` | Example shell script (template) |

## Notes

- Cloud-init runs **before** Packer provisioners
- Use `cloud-init status --wait` in provisioners to ensure completion
- User data must be valid YAML for `#cloud-config`
- Maximum user data size: 16KB (base64 encoded)

## Debugging

Check cloud-init logs on the VM:

```bash
# Cloud-init output
sudo cat /var/log/cloud-init-output.log

# Cloud-init status
cloud-init status --long

# Generated files
ls -la /opt/webapp/
```
