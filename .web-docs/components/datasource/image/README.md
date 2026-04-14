Type: `numspot-bsu`

The `numspot-bsu` image datasource resolves a Numspot image by filter **before** the build starts. It is useful when you don't want to hardcode an image ID in your template — for example, in CI/CD pipelines where a base image is built in a previous job and referenced by name in a subsequent job.

## How It Works

1. The datasource queries the Numspot API for images matching the given filters.
2. If `most_recent = true`, the image with the most recent creation date is selected.
3. If no image is found, Packer fails immediately — no VM is created.
4. The resolved image attributes are available as outputs (e.g. `data.numspot-bsu.image.base.id`).

## Configuration

### Required

At least one entry in `filters` must be provided.

| Option | Type | Description |
|--------|------|-------------|
| `client_id` | string | OAuth2 client ID (UUID). Can use `NUMSPOT_CLIENT_ID` env var. |
| `client_secret` | string | OAuth2 client secret. Can use `NUMSPOT_CLIENT_SECRET` env var. |
| `space_id` | string | Numspot space ID (UUID). Can use `NUMSPOT_SPACE_ID` env var. |
| `filters` | map(string) | Key/value map of filter criteria. At least one entry required. |

### Optional

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `region` | string | `eu-west-2` | Numspot region. |
| `most_recent` | bool | `false` | If multiple images match, select the most recently created one. |
| `owners` | list(string) | `[]` | Filter images by owner account IDs. Optional — your space already scopes image visibility. |

### Filter Keys

| Key | Description |
|-----|-------------|
| `name` | Image name (exact match) |
| `id` | Image ID (e.g. `ami-bf56f9be`) |
| `state` | Image state (`available`, `pending`, `failed`) |
| `architecture` | Architecture (e.g. `x86_64`) |
| `description` | Image description |
| `root_device_type` | Root device type |
| `root_device_name` | Root device name |

## Output Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Image ID (e.g. `ami-bf56f9be`) |
| `name` | string | Image name |
| `description` | string | Image description |
| `creation_date` | string | Creation date in ISO 8601 format |
| `state` | string | Image state (`available`, `pending`, `failed`) |
| `architecture` | string | Architecture (e.g. `x86_64`) |
| `root_device_name` | string | Root device name |
| `root_device_type` | string | Root device type (always `bsu`) |
| `tags` | map(string) | Map of tags assigned to the image |

## Example: Resolve by Name

```hcl
packer {
  required_plugins {
    numspot = {
      version = ">= 0.1.0"
      source  = "github.com/numspot/numspot"
    }
  }
}

data "numspot-bsu" "image" "base" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  filters = {
    name = "ubuntu-22.04-base"
  }
  most_recent = true
}

source "numspot-bsu" "example" {
  client_id     = var.client_id
  client_secret = var.client_secret
  space_id      = var.space_id

  source_image = data.numspot-bsu.image.base.id
  image_name   = "my-app-${formatdate("YYYY-MM-DD", timestamp())}"
  vm_type      = "ns-eco7-2c2r"
  ssh_username = "ubuntu"

  tags = {
    SourceImage = data.numspot-bsu.image.base.name
  }
}

build {
  sources = ["source.numspot-bsu.example"]
}
```

## Example: Chained CI/CD Pipeline

A common pattern is to chain two Packer builds: the first creates a base image, the second finds it by name and adds application layers.

**Job 1 — build base image:**
```hcl
source "numspot-bsu" "base" {
  source_image = "ami-54a37c4a"   # upstream OS image
  image_name   = "my-base-${formatdate("YYYYMMDD", timestamp())}"
  # ...
}
```

**Job 2 — build application image (no hardcoded ID):**
```hcl
data "numspot-bsu" "image" "base" {
  filters     = { name = "my-base-20260101" }
  most_recent = true
  # ...
}

source "numspot-bsu" "app" {
  source_image = data.numspot-bsu.image.base.id
  # ...
}
```

## Notes

- Unlike the Outscale equivalent, `owners` is **optional**. Numspot images are scoped to your space, so all visible images already belong to your account.
- Exact name matching is required — wildcards are not supported in the `name` filter.
- Use `most_recent = true` when multiple images may match (e.g. daily builds with timestamps in the name) to always pick the latest one.
