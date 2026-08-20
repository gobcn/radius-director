# Docker Deployment

This guide describes how to initialize, configure, generate, and operate a RADIUS Director deployment using Docker.

RADIUS Director is designed to run entirely in Docker. No RADIUS Director installation on the host is required.

The host only needs Docker and Docker Compose. The RADIUS Director CLI is run from the RADIUS Director Docker image.

RADIUS Director uses two separate Docker Compose environments:

1. The **RADIUS Director runtime**, which runs the RADIUS Director CLI and provides access to the runtime configuration, assets, and generated files.
2. The **generated deployment**, which contains the RADIUS server and database services generated from the RADIUS Director configuration.

Keeping these two environments separate allows RADIUS Director itself to be upgraded independently from the services it manages.

## Prerequisites

The host should have Docker installed and be able to run Docker containers.

No RADIUS Director executable needs to be installed on the host.

The commands in this guide use the RADIUS Director Docker image:

```text
gobcn/radius-director:latest
```

The examples in this guide assume a Linux host and use `/opt/radius-director` as the runtime directory.

## Runtime Directory

A RADIUS Director runtime is a directory containing the configuration, assets, generated deployment, and Docker Compose configuration needed to operate a deployment.

For a Linux Docker deployment, `/opt/radius-director` is recommended as the runtime directory.

Create the directory and give the current user ownership:

```bash
sudo mkdir -p /opt/radius-director
sudo chown "$USER":"$USER" /opt/radius-director
```

A typical runtime directory looks like this:

```text
/opt/radius-director/
├── assets/
│   ├── schemas/
│   └── templates/
├── config/
│   └── example.yaml
├── generated/
├── .env
└── compose.yaml
```

The directories have the following purposes:

| Directory | Purpose |
|---|---|
| `assets/` | Templates and schemas used by RADIUS Director |
| `config/` | RADIUS Director configuration files |
| `generated/` | Generated deployment files |
| `.env` | Runtime environment settings |
| `compose.yaml` | Docker Compose configuration for the RADIUS Director runtime |

The `assets/templates` and `assets/schemas` directories are initially empty. The assets shipped with RADIUS Director are exported separately so that they can be reviewed or customized before being used by the deployment.

## Initialize a Runtime

The runtime is initialized using the RADIUS Director Docker image.

The `init` command has the following syntax:

```text
radius-director init <runtime-directory> <network-name>
```

The `init` command creates the Docker network specified by `<network-name>`, so the RADIUS Director container needs access to the host Docker daemon during initialization. The Docker socket is mounted into the container for this purpose. It is not required for normal RADIUS Director CLI operations.

The container should also run as the host user during initialization. This ensures that the runtime files and directories created on the host are owned by the user who will manage the deployment.

For a Docker-based deployment, mount the directory that will contain the runtime into the container as `/workspace`:

```bash
docker run --rm \
  --user "$(id -u):$(id -g)" \
  --group-add "$(getent group docker | cut -d: -f3)" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /opt/radius-director:/workspace \
  gobcn/radius-director:latest \
  init /workspace radius-director
```

In this example:

- `/workspace` is the runtime directory as seen from inside the container.
- `/opt/radius-director` is the corresponding runtime directory on the host.
- `radius-director` is the name of the Docker network that will be used by the generated deployment.
- `id -u` and `id -g` provide the current user's UID and primary GID. These are used so that files created in the runtime directory are owned by the host user.
- `getent group docker | cut -d: -f3` obtains the GID of the host's `docker` group. This is supplied as a supplementary group so that `init` can access the mounted Docker socket.

The command creates the runtime structure in `/opt/radius-director`.

The network is created by the `init` command if it does not already exist.

The resulting `.env` contains values similar to:

```text
RADIUS_DIRECTOR_RUNTIME_NETWORK=radius-director
RADIUS_DIRECTOR_UID=1000
RADIUS_DIRECTOR_GID=1000
```

The UID and GID values correspond to the user that ran the `init` command. They are used by the runtime's `compose.yaml` so that subsequent RADIUS Director containers run with the same host user identity and can write to the runtime's bind-mounted directories.

After initialization, change to the runtime directory:

```bash
cd /opt/radius-director
```

From this point onward, the runtime's `compose.yaml` can be used to run RADIUS Director commands. The Docker socket is not required for these normal operations.

### Runtime initialization does not export assets

The `init` command creates the asset directories, but does not populate them with the bundled templates and schemas.

This is intentional. Exporting the assets is a separate operation so that the shipped assets can be inspected and compared with locally customized assets.

## Export the Bundled Assets

Once the runtime has been initialized, export the templates and schemas shipped with RADIUS Director:

```bash
docker compose \
  run --rm radius-director export assets
```

The runtime Compose file provides the configured asset location to the RADIUS Director container, so the destination does not need to be specified for the normal runtime workflow.

The exported assets will be placed in:

```text
/opt/radius-director/
└── assets/
    ├── schemas/
    └── templates/
```

The export operation will not overwrite existing non-empty asset directories.

This protects customized templates and schemas from accidentally being overwritten.

### Exporting assets to another location

The `export assets` command also accepts an explicit destination:

```text
radius-director export assets <output-directory>
```

This is useful when inspecting the assets shipped with a new RADIUS Director version without modifying the assets currently used by a deployment.

For example, create a separate directory and mount it into the container:

```bash
mkdir -p /opt/radius-director/new-assets

docker compose \
  run --rm \
  -v /opt/radius-director/new-assets:/export \
  radius-director export assets /export
```

The new assets will then be available on the host in:

```text
/opt/radius-director/new-assets/
├── schemas/
└── templates/
```

This is particularly useful when upgrading RADIUS Director. The newly shipped assets can be compared with locally customized assets and changes can be merged manually as appropriate.

## Configure RADIUS Director

The runtime configuration is stored in the `config` directory.

After exporting the assets, review the example configuration:

```text
/opt/radius-director/config/example.yaml
```

The configuration defines the global objects and tenants used by the deployment.

A configuration contains sections such as:

```yaml
global_objects:
  credential_profiles:
    default:
      shared_secret: shared-secret

  authentication_profiles:
    default:
      simultaneous_use: 1

  accounting_profiles:
    default: {}

  monitoring_profiles:
    default: {}

  deployment_profiles:
    default:
      template: default
      overlays: []

  nas_devices:
    mt-core-01.gobcn.ca:
      ip_address: 10.10.10.1
      vendor: mikrotik

tenants:
  customer-a:
    authentication_profile: default
    deployment_profile: default
    ...
```

The example configuration should be treated as a starting point. Create or modify a configuration appropriate for the deployment.

The configuration file can have any filename. For example:

```text
/opt/radius-director/config/production.yaml
```

## Validate the Configuration

Before generating the deployment, validate the configuration.

From inside the runtime container, configuration files are available under `/config`.

For example:

```bash
docker compose \
  run --rm radius-director validate /config/production.yaml
```

A successful validation indicates that the configuration can be loaded and validated against the available templates and schemas.

If validation fails, correct the configuration before attempting to generate the deployment.

## Generate the Deployment

Once the configuration has been validated, generate the deployment:

```bash
docker compose \
  run --rm radius-director generate /config/production.yaml /generated
```

The generated files are written to the runtime's `generated` directory.

For example:

```text
/opt/radius-director/
└── generated/
    ├── docker-compose.yml
    ├── entrypoint.sh
    └── ...
```

The generated `docker-compose.yml` contains the services required by the configured tenants.

The generated Compose file also references the runtime Docker network.

The network name is resolved when RADIUS Director generates the file. The generated deployment therefore does not depend on the runtime `.env` file being present when the generated deployment is subsequently started.

## Start the Generated Deployment

The generated deployment is started separately from the RADIUS Director runtime.

From `/opt/radius-director`:

```bash
docker compose \
  -f ./generated/docker-compose.yml \
  up -d
```

The generated Compose file will start the services required by the configured deployment.

For example, a tenant may result in:

```text
radius-customer-a
database-customer-a
```

The services are attached to the runtime's external Docker network.

You can check the generated deployment with:

```bash
docker compose \
  -f ./generated/docker-compose.yml \
  ps
```

You can also inspect the Docker network directly:

```bash
docker network inspect radius-director
```

The generated RADIUS and database containers should appear as members of the configured network.

## Runtime Compose vs. Generated Compose

It is important to understand the distinction between the two Compose files.

### Runtime Compose

```text
/opt/radius-director/compose.yaml
```

This runs RADIUS Director itself.

It provides access to:

```text
Host                         Container
────────────────────────────────────────────
./assets          →         /assets
./config          →         /config
./generated       →         /generated
```

The runtime Compose also provides the RADIUS Director runtime network configuration.

Commands such as `validate`, `generate`, `export assets`, and `maintenance accounting` are run using this Compose file.

### Generated Compose

```text
/opt/radius-director/generated/docker-compose.yml
```

This is the deployment generated by RADIUS Director.

It contains the actual services that make up the RADIUS deployment, such as FreeRADIUS and its database.

The generated deployment is independent of the RADIUS Director runtime container and can continue running after a RADIUS Director CLI container exits.

## Managing the Generated Deployment

The generated deployment can be managed using normal Docker Compose commands.

View the running services:

```bash
docker compose \
  -f ./generated/docker-compose.yml \
  ps
```

Start the deployment:

```bash
docker compose \
  -f ./generated/docker-compose.yml \
  up -d
```

Stop the deployment:

```bash
docker compose \
  -f ./generated/docker-compose.yml \
  down
```

View logs:

```bash
docker compose \
  -f ./generated/docker-compose.yml \
  logs
```

Logs for a particular service can also be viewed:

```bash
docker compose \
  -f ./generated/docker-compose.yml \
  logs radius-customer-a
```

## Accounting Maintenance

RADIUS Director provides accounting maintenance operations that are intended to run periodically.

Because accounting maintenance operates against the deployment database, it can be run using the RADIUS Director runtime container.

For example, to run accounting maintenance manually for the `customer-a` tenant:

```bash
docker compose \
  run --rm radius-director maintenance accounting /config/production.yaml customer-a
```

The first argument identifies the configuration file and the second identifies the tenant whose accounting data should be maintained.

The runtime container has access to the configuration and can communicate with the deployment database using the configured database connection.

### Scheduling Accounting Maintenance

Accounting maintenance should be run regularly to keep accounting and session state current, including cleaning up stale sessions.

On Linux, this can be scheduled using the `cron` service.

For example, to run accounting maintenance every five minutes for `customer-a`, edit the crontab for the user that owns the RADIUS Director runtime:

```bash
crontab -e
```

Add:

```cron
*/5 * * * * cd /opt/radius-director && /usr/bin/flock -n /tmp/radius-director-accounting-customer-a.lock docker compose run --rm radius-director maintenance accounting /config/production.yaml customer-a >> /var/log/radius-director/accounting-customer-a.log 2>&1
```

The command:

- Changes to `/opt/radius-director` so Docker Compose uses the runtime's `compose.yaml` and `.env`.
- Runs the maintenance operation every five minutes.
- Uses `flock` to prevent another accounting maintenance run from starting if the previous run is still in progress.
- Runs the RADIUS Director accounting maintenance command in a temporary container.
- Appends output and errors to a log file under `/var/log/radius-director/`.

Create the log directory before enabling the scheduled job:

```bash
sudo mkdir -p /var/log/radius-director
sudo chown "$USER":"$USER" /var/log/radius-director
```

If multiple tenants require accounting maintenance, create a separate cron entry for each tenant. For example:

```cron
*/5 * * * * cd /opt/radius-director && /usr/bin/flock -n /tmp/radius-director-accounting-customer-a.lock docker compose run --rm radius-director maintenance accounting /config/production.yaml customer-a >> /var/log/radius-director/accounting-customer-a.log 2>&1
*/5 * * * * cd /opt/radius-director && /usr/bin/flock -n /tmp/radius-director-accounting-customer-b.lock docker compose run --rm radius-director maintenance accounting /config/production.yaml customer-b >> /var/log/radius-director/accounting-customer-b.log 2>&1
```

The `flock` lock is per tenant, so maintenance for different tenants can run independently.

The schedule can be adjusted as appropriate for the deployment, but a five-minute interval is a suitable starting point for normal session maintenance.

To verify that the scheduled jobs are present:

```bash
crontab -l
```

To review the output from a scheduled maintenance job:

```bash
tail -f /var/log/radius-director/accounting-customer-a.log
```

### Log Rotation

Because the accounting job runs every five minutes, its output can grow over time.

The logs are stored under `/var/log/radius-director/` so that they can be managed using the Linux `logrotate` system.

Create:

```text
/etc/logrotate.d/radius-director
```

with:

```text
/var/log/radius-director/*.log {
    weekly
    rotate 12
    compress
    delaycompress
    missingok
    notifempty
}
```

This example keeps twelve weeks of logs, compressing older log files.

The cron job should run as the same Linux user that has permission to access the RADIUS Director runtime and Docker. If Docker requires elevated privileges on the host, configure the appropriate Docker permissions for the account running the cron job rather than adding `sudo` to the cron command.

## Updating RADIUS Director

When upgrading to a new RADIUS Director version, do not automatically overwrite customized templates or schemas.

A recommended upgrade process is:

1. Stop or otherwise prepare the existing deployment according to your operational requirements.
2. Update the RADIUS Director image.
3. Export the new version's bundled assets to a separate directory.
4. Compare the new assets with the customized assets currently used by the deployment.
5. Merge any required changes into the customized assets.
6. Validate the configuration.
7. Generate the deployment using the updated assets.
8. Review the generated deployment.
9. Start or update the generated deployment.

For example, export the new version's assets to a temporary directory:

```bash
mkdir -p /opt/radius-director/new-assets

docker compose \
  run --rm \
  -v /opt/radius-director/new-assets:/export \
  radius-director export assets /export
```

This does not modify the existing:

```text
/opt/radius-director/assets/
```

directory.

The administrator can then compare:

```text
/opt/radius-director/assets/
```

with:

```text
/opt/radius-director/new-assets/
```

and merge changes as necessary.

After the comparison is complete, the temporary directory can be removed:

```bash
rm -rf /opt/radius-director/new-assets
```

## Custom Templates and Schemas

Templates and schemas are stored in the runtime's `assets` directory.

These files can be customized for the deployment.

Because the assets are mounted into the RADIUS Director runtime container, changes made on the host are immediately available to RADIUS Director operations performed through Docker Compose.

Customized assets should be preserved when upgrading RADIUS Director.

When a new RADIUS Director version provides updated templates or schemas, export the new bundled assets to a separate directory first and compare them with the customized versions.

Do not use an asset export as a mechanism for blindly overwriting customized assets.

## Docker Network

The runtime is associated with a Docker network specified when the runtime is initialized.

For example:

```bash
docker run --rm \
  -v /opt/radius-director:/workspace \
  gobcn/radius-director:latest \
  init /workspace radius-director
```

creates or uses the Docker network:

```text
radius-director
```

The runtime's `.env` contains:

```text
RADIUS_DIRECTOR_RUNTIME_NETWORK=radius-director
```

The generated deployment uses this network as an external Docker network.

The RADIUS services and their associated database services therefore communicate over the same Docker network.

The network is deliberately external to the generated Compose project. This allows the runtime and generated deployment to share the same network without either Compose project owning the other's network lifecycle.

## Host and Container Paths

When RADIUS Director is run through Docker Compose, paths supplied to the CLI refer to paths inside the container, not paths on the host.

For example:

```text
/config/production.yaml
```

refers to the host file:

```text
/opt/radius-director/config/production.yaml
```

because the runtime Compose mounts:

```text
./config:/config
```

Similarly:

```text
/generated
```

refers to the host directory:

```text
/opt/radius-director/generated/
```

because the runtime Compose mounts:

```text
./generated:/generated
```

The normal mapping is:

| Host | Container |
|---|---|
| `./assets` | `/assets` |
| `./config` | `/config` |
| `./generated` | `/generated` |

This distinction is important when running RADIUS Director commands through Docker Compose.

## Troubleshooting

### The runtime network already exists

If `init` reports that a network belongs to a different RADIUS Director runtime, do not reuse that network unless it is intentionally associated with the runtime being initialized.

A RADIUS Director-managed network contains a runtime identifier in its Docker labels.

Inspect the network with:

```bash
docker network inspect <network-name>
```

If necessary, choose a different network name.

### `export assets` refuses to overwrite a directory

RADIUS Director intentionally refuses to export over a non-empty `templates` or `schemas` directory.

This protects customized assets.

If you want to inspect the assets shipped with a new version, export them to a separate directory instead:

```bash
mkdir -p /opt/radius-director/new-assets

docker compose \
  run --rm \
  -v /opt/radius-director/new-assets:/export \
  radius-director export assets /export
```

### The generated deployment cannot find the external network

Verify that the network exists:

```bash
docker network ls
```

Then inspect it:

```bash
docker network inspect radius-director
```

The network name in the generated Compose file should match the network created for the runtime.

### A configuration file cannot be found

Remember that commands executed inside the RADIUS Director container use container paths.

For example, this:

```text
/config/production.yaml
```

corresponds to:

```text
/opt/radius-director/config/production.yaml
```

on the host.

### Generated files are not appearing on the host

The runtime Compose mounts:

```text
./generated:/generated
```

so generated files written to `/generated` inside the container should appear in the runtime's `generated` directory on the host.

Verify that the command is being run with the runtime `compose.yaml`, rather than the generated deployment Compose file.

## Summary

The normal Docker-based workflow is:

```text
1. Create the runtime directory
       ↓
2. Initialize the runtime using Docker
       ↓
3. Export the bundled assets
       ↓
4. Configure the deployment
       ↓
5. Validate the configuration
       ↓
6. Generate the deployment
       ↓
7. Start the generated Compose deployment
       ↓
8. Run accounting maintenance as required
```

A typical initial setup therefore looks like:

```bash
# Create the runtime directory
sudo mkdir -p /opt/radius-director
sudo chown "$USER":"$USER" /opt/radius-director

# Initialize the runtime
docker run --rm \
  -v /opt/radius-director:/workspace \
  gobcn/radius-director:latest \
  init /workspace radius-director

# Enter the runtime directory
cd /opt/radius-director

# Export bundled templates and schemas
docker compose \
  run --rm radius-director export assets

# Edit configuration
# /opt/radius-director/config/production.yaml

# Validate
docker compose \
  run --rm radius-director validate /config/production.yaml

# Generate
docker compose \
  run --rm radius-director generate /config/production.yaml /generated

# Start generated deployment
docker compose \
  -f ./generated/docker-compose.yml \
  up -d
```

The RADIUS Director runtime and the generated deployment are intentionally separate. The runtime provides the tools and configuration used to build and maintain the deployment, while the generated Compose project runs the actual RADIUS services.