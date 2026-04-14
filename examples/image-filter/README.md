# Example: Image Filter Datasource

This example shows how to use the `numspot-bsu` datasource to resolve a source image
dynamically instead of hardcoding an image ID.

## What it does

1. The datasource queries the Numspot API for an image matching the given name **before** the build starts.
2. If no image is found, Packer fails immediately — no VM is created.
3. The resolved image ID is injected into the builder via `data.numspot-bsu.image.base.id`.

## Usage

```bash
export NUMSPOT_CLIENT_ID="<your-client-id>"
export NUMSPOT_CLIENT_SECRET="<your-client-secret>"
export NUMSPOT_SPACE_ID="<your-space-id>"

packer build \
  -var "client_id=$NUMSPOT_CLIENT_ID" \
  -var "client_secret=$NUMSPOT_CLIENT_SECRET" \
  -var "space_id=$NUMSPOT_SPACE_ID" \
  -var "subnet_id=<your-subnet-id>" \
  -var "source_image_name=<exact-image-name>" \
  .
```

## Available datasource outputs

| Field | Description |
|---|---|
| `id` | Image ID (e.g. `ami-bf56f9be`) |
| `name` | Image name |
| `description` | Image description |
| `creation_date` | Creation date (ISO 8601) |
| `state` | Image state (`available`, `pending`, `failed`) |
| `architecture` | Architecture (e.g. `x86_64`) |
| `root_device_name` | Root device name |
| `root_device_type` | Root device type (always `bsu`) |
| `tags` | Map of tags |

## When is this useful?

The datasource shines in **chained CI/CD pipelines**: a first job builds a base image,
a second job finds the latest version of that base image by name — without passing
the image ID between jobs.
