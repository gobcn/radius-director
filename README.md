# RADIUS Director

RADIUS Director helps you configure and deploy FreeRADIUS servers from a single YAML configuration. If you manage RADIUS for multiple customers or environments and are tired of copying configuration files, tracking manual changes, or keeping servers consistent, it gives you a repeatable way to describe and deploy them.

Define shared settings such as NAS devices and credential profiles once, then reference them from each tenant. RADIUS Director validates the configuration and generates a separate FreeRADIUS configuration and Docker deployment for each tenant. The same input produces the same generated configuration, making changes easier to review and reproduce.

FreeRADIUS still handles authentication, authorization, accounting, and RADIUS traffic. RADIUS Director manages the deployment-specific configuration around it; it does not replace FreeRADIUS or your billing system.

## How it works

1. Describe your shared objects and tenants in YAML, including each tenant's database, NAS assignments, and FreeRADIUS version.
2. Validate the configuration to catch errors before deployment.
3. Generate isolated tenant configuration trees and a Docker Compose deployment using version-compatible templates.
4. Start the generated deployment and maintain it with normal Docker Compose commands.

The [example configuration](resources/example.yaml) shows the supported configuration structure. Generated files are artifacts: make changes in the source configuration and regenerate them instead of editing them by hand.

## Get started

The supported deployment uses Docker and Docker Compose. Follow the [Docker deployment guide](docs/docker-deployment.md) to initialize a runtime, configure a tenant, and start a deployment. It also covers operations, accounting maintenance, upgrades, customization, and troubleshooting.

## Project status

RADIUS Director is in early development. Configuration loading and validation, deterministic generation, tenant configuration and manifests, Docker Compose deployment, and accounting maintenance are implemented. Development is focused on deployment lifecycle and operational tooling.

## Learn more

- [Vision and goals](docs/VISION.md)
- [Architecture and tenant model](docs/ARCHITECTURE.md)
- [Configuration schema](docs/CONFIGURATION_SCHEMA.md) and [validation rules](docs/VALIDATION.md)
- [Generation pipeline](docs/GENERATOR.md) and [managed configuration ownership](docs/adr/ADR-0005-managed-configuration-tree-ownership.md)

Licensed under the [Apache License 2.0](LICENSE).
