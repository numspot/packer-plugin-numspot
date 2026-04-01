# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.0] - 2026-04-01

### Added

- Initial release of Numspot Packer plugin
- `numspot-bsu` builder for creating Numspot Images from source images
- OAuth2 authentication with automatic token refresh
- Support for all Numspot VM types (`ns-eco7-2c2r`, `ns-eco7-4c4r`, etc.)
- Automatic subnet discovery or explicit subnet configuration
- Security group auto-creation with configurable CIDR
- Public IP association for SSH access
- Block device mapping support (launch and image)
- Tag support for images, VMs, volumes, and snapshots
- Cloud-init user data support
- Source image filtering
- Comprehensive documentation and examples

### Known Limitations

- Only BSU builder implemented (chroot, bsusurrogate, bsuvolume builders not yet available)
- No datasource component (image lookup in HCL)

### Dependencies

- Go 1.24+
- Packer SDK v0.6+
- Numspot SDK (generated from OpenAPI spec)
