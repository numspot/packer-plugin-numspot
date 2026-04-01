# Troubleshooting Guide

This guide covers common issues and their solutions when using the Numspot Packer plugin.

## Table of Contents

- [Authentication Errors](#authentication-errors)
- [Permission Errors](#permission-errors)
- [Network Errors](#network-errors)
- [VM Errors](#vm-errors)
- [SSH Errors](#ssh-errors)
- [Image Errors](#image-errors)

---

## Authentication Errors

### Error: 401 Unauthorized

**Cause:** Invalid or expired OAuth2 credentials.

**Solution:**
1. Verify `client_id` and `client_secret` are correct
2. Check that credentials haven't been revoked in your Numspot console
3. Ensure the `host` URL is correct (use `.com` not `.internal`)

```
Error: error creating numspot client: authentication failed: 401 Unauthorized
```

### Error: client_id is required

**Cause:** OAuth2 client ID not provided.

**Solution:**
Set `client_id` in your Packer template or via environment variable:

```bash
export NUMSPOT_CLIENT_ID="af727e6f-a900-41f0-b83e-b5614f9ac00a"
```

Or in HCL:

```hcl
source "numspot-bsu" "example" {
  client_id = "af727e6f-a900-41f0-b83e-b5614f9ac00a"
  # ...
}
```

### Error: Invalid client_id format

**Cause:** `client_id` is not a valid UUID.

**Solution:**
Ensure `client_id` is in UUID format (8-4-4-4-12):

```
Valid:   af727e6f-a900-41f0-b83e-b5614f9ac00a
Invalid: my-client-id
Invalid: af727e6fa90041f0b83eb5614f9ac00a
```

---

## Permission Errors

### Error: 403 Forbidden

**Cause:** Service account lacks required permissions.

**Solution:**
Grant `compute.all` permission to your service account on the specified space.

Common 403 errors:

| API Call | Error Message | Solution |
|----------|---------------|----------|
| ReadImages | 403 on ReadImages | Grant read permissions |
| CreateVm | 403 on CreateVm | Grant compute permissions |
| CreateKeypair | 403 on CreateKeypair | Grant compute permissions |
| CreateImage | 403 on CreateImage | Grant compute permissions |

The `compute.all` permission covers all operations needed by the plugin.

### Error: space_id is required

**Cause:** Space ID not provided.

**Solution:**
Set `space_id` in your Packer template or via environment variable:

```bash
export NUMSPOT_SPACE_ID="673bb25a-294d-4ded-b1dd-b63a7d701ea6"
```

---

## Network Errors

### Error: VpcId is required but not found

**Cause:** No subnet found and no `net_id` or `subnet_id` specified.

**Solution:**
Either:
1. Create a subnet in your VPC before running Packer
2. Specify `subnet_id` explicitly:

```hcl
source "numspot-bsu" "example" {
  subnet_id = "subnet-0111f5de"
  # ...
}
```

3. Or use filters to discover a subnet:

```hcl
subnet_filter {
  name   = "tag:Environment"
  values = ["production"]
}
```

### Error: SSH timeout / Connection refused

**Cause:** VM not accessible via SSH.

**Solutions:**

1. **Check Internet Gateway is attached:**
   - VPC must have IGW attached
   - Route table must have route `0.0.0.0/0` → IGW

2. **Check security group:**
   - Plugin creates temporary security group allowing port 22
   - If using custom `security_group_ids`, ensure SSH (port 22) is allowed

3. **Check VM state:**
   - VM must be in `running` state
   - Wait for cloud-init to complete (Rancher images may take 1-2 minutes)

4. **Verify SSH interface:**

```hcl
# Use public IP (default)
ssh_interface = "public_ip"

# Or use private IP (if on VPN/Direct Connect)
ssh_interface = "private_ip"
```

### Error: Public IP not accessible

**Cause:** Internet Gateway not configured correctly.

**Solution:**
1. Attach Internet Gateway to VPC
2. Add route `0.0.0.0/0` → IGW in route table
3. Ensure `associate_public_ip_address = true` (default)

---

## VM Errors

### Error: Invalid VM type 'tinav5.c1r1p1'

**Cause:** Using Outscale VM type on Numspot.

**Solution:**
Use Numspot VM types:

```hcl
# Correct
vm_type = "ns-eco7-2c2r"

# Wrong (Outscale type)
vm_type = "tinav5.c1r1p1"
```

Valid Numspot VM types:
- `ns-eco7-2c2r`
- `ns-eco7-4c4r`

### Error: vm_type is required

**Cause:** VM type not specified.

**Solution:**
Add `vm_type` to your configuration:

```hcl
source "numspot-bsu" "example" {
  vm_type = "ns-eco7-2c2r"
  # ...
}
```

### Error: VM stuck in pending/creating state

**Cause:** Insufficient resources or quota exceeded.

**Solutions:**
1. Check your Numspot quota limits
2. Try a different VM type
3. Check for resource contention in the region

### Error: Boot mode not supported

**Cause:** Invalid boot mode specified.

**Solution:**
Use valid boot modes:

```hcl
# Valid options
boot_mode = "legacy"
boot_mode = "uefi"
```

---

## SSH Errors

### Error: SSH authentication failed

**Cause:** Wrong SSH username for the source image.

**Solution:**
Use the correct username for your image:

| Image Type | Username |
|------------|----------|
| Rancher/Outscale (Debian 10) | `outscale` |
| Ubuntu | `ubuntu` |
| Debian | `debian` |
| CentOS | `centos` |

Example:

```hcl
# For Rancher/Outscale images
ssh_username = "outscale"

# For Ubuntu images  
ssh_username = "ubuntu"
```

### Error: Permission denied (publickey)

**Cause:** SSH key not configured correctly.

**Solutions:**

1. **Let plugin create temporary keypair (default):**
   ```hcl
   # No configuration needed - plugin auto-generates keypair
   ```

2. **Use existing keypair:**
   ```hcl
   ssh_keypair_name     = "my-keypair"
   ssh_private_key_file = "~/.ssh/id_rsa"
   ```

### Error: Handshake failed

**Cause:** SSH protocol version mismatch.

**Solution:**
Ensure Packer is configured to use SSH:

```hcl
communicator = "ssh"
ssh_username = "outscale"
```

---

## Image Errors

### Error: Error creating image: no Id returned

**Cause:** Image creation API call succeeded but no image ID was returned.

**Solution:**
1. Check API permissions (`compute.all` required)
2. Verify `image_name` is valid (3-128 chars, alphanumeric/hyphens/underscores)

### Error: Source image not found

**Cause:** Invalid `source_image` ID or filter.

**Solutions:**

1. **Verify image ID exists:**
   ```hcl
   source_image = "ami-52b3214f"  # Check this ID exists in your space
   ```

2. **Use image filter:**
   ```hcl
   source_image_filter {
     name        = "name"
     values      = ["ubuntu-22.04-*"]
     owners      = ["numspot"]
     most_recent = true
   }
   ```

3. **Check image is in the correct space:**
   - Images are scoped to spaces
   - Ensure you're using the right `space_id`

### Error: Image name already exists

**Cause:** Image with the same name already exists.

**Solutions:**

1. **Use unique name with timestamp:**
   ```hcl
   image_name = "my-app-${timestamp()}"
   ```

2. **Force deregister existing image:**
   ```hcl
   force_deregister = true
   ```

### Error: image_name contains invalid characters

**Cause:** Invalid characters in image name.

**Solution:**
Image name must:
- Be 3-128 characters
- Contain only alphanumeric characters, hyphens, and underscores

```hcl
# Valid
image_name = "my-app-image_2024"

# Invalid (contains spaces)
image_name = "my app image"

# Invalid (contains dots)
image_name = "my.app.image"
```

---

## General Debugging

### Enable Debug Logging

```bash
# Enable Packer debug logging
export PACKER_LOG=1
export PACKER_LOG_PATH=packer.log

packer build template.pkr.hcl
```

### Run with OnError

```bash
# Keep VM on failure for debugging
packer build -on-error=abort template.pkr.hcl
```

### Check VM State

If the build fails, check the VM state in Numspot console:
1. Navigate to Compute → Virtual Machines
2. Filter by name containing "packer"
3. Check:
   - VM state (running, stopped, error)
   - Console output
   - Security groups attached

### Common Issues Checklist

- [ ] `client_id`, `client_secret`, `space_id`, `host` all set correctly
- [ ] Service account has `compute.all` permission
- [ ] VPC has Internet Gateway attached
- [ ] Route table has `0.0.0.0/0` → IGW
- [ ] Subnet exists in VPC
- [ ] Using Numspot VM type (not Outscale type)
- [ ] Correct SSH username for source image
- [ ] `source_image` ID exists in the space
- [ ] `image_name` is unique and valid

---

## Getting Help

If you continue to experience issues:

1. Check the [Configuration Reference](./configuration.md)
2. Review the [README](../README.md) for prerequisites
3. Check the [Numspot documentation](https://docs.numspot.com)
4. Open an issue with:
   - Full error message
   - Packer template (sanitized of credentials)
   - Debug log output
