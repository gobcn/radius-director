# Configuration Schema

This document defines the declarative configuration format used by RADIUS Director.

The schema described here is independent of any implementation language.

---

# Goals

The configuration format should be:

- human readable
- version controllable
- easy to review
- deterministic
- vendor neutral

YAML is currently the preferred configuration format.

---

# Root Configuration

A RADIUS Director configuration consists of a single root configuration document.

The root contains two top-level collections:

- Global Objects
- Tenants

```yaml
global_objects:

  credential_profiles:

  authentication_profiles:

  accounting_profiles:

  monitoring_profiles:

  nas_devices:

  trusted_radius_clients:

tenants:
```

---

# Global Objects

Global Objects are defined once and may be referenced by one or more tenants.

The following Global Object collections are supported:

| Collection | Object |
|------------|--------|
| credential_profiles | Credential Profile |
| authentication_profiles | Authentication Profile |
| accounting_profiles | Accounting Profile |
| monitoring_profiles | Monitoring Profile |
| nas_devices | NAS Device |
| trusted_radius_clients | Trusted RADIUS Client |

Each object is identified by its YAML key.

Example:

```yaml
global_objects:

  credential_profiles:

    default:
      shared_secret: mysecret

  accounting_profiles:

    default:
      stale_session_timeout: 20m

  nas_devices:

    mt-core-01.gobcn.ca:
      ip_address: 10.10.10.1
      vendor: mikrotik

  trusted_radius_clients:

    sonar:
      ip_address: 20.104.33.4
```

---

# Authentication Profiles

Authentication Profiles define reusable tenant-wide subscriber authentication policy.

Example:

```yaml
authentication_profiles:

  default:
    simultaneous_use: 1
```

Each tenant references one Authentication Profile using `authentication_profile`.

The `simultaneous_use` property defines the default maximum number of simultaneous subscriber sessions.

---

# Accounting Profiles

Accounting Profiles define reusable accounting behaviour.

An Accounting Profile may define a `stale_session_timeout`.

Example:

```yaml
accounting_profiles:

  default:
    stale_session_timeout: 20m
```

The `stale_session_timeout` property defines how long a session may remain without accounting activity before it is considered stale.

The value is expressed as a duration.

Examples include:

```yaml
stale_session_timeout: 10m
stale_session_timeout: 20m
stale_session_timeout: 1h
```

If `stale_session_timeout` is omitted, automatic stale-session cleanup is not enabled for that Accounting Profile.

The timeout defines when a session is considered stale. It does not define how frequently stale-session maintenance is executed.

When a stale session is closed, its recorded stop time represents the last known accounting activity for the session rather than the time at which the cleanup process executes.

The `stale_session_timeout` applies independently to each NAS Assignment that references the Accounting Profile.

Different NAS Assignments within the same tenant may reference Accounting Profiles with different `stale_session_timeout` values.

This allows stale-session policy to reflect differences in accounting behaviour between NAS Devices, including differences in their configured interim accounting update intervals.

A NAS Assignment referencing an Accounting Profile without `stale_session_timeout` does not participate in automatic stale-session cleanup.

---

# Tenants

The `tenants` collection contains one or more Tenant objects.

Each tenant is identified by its YAML key.

Example:

```yaml
tenants:

  customer-a:

    authentication_profile: default

    database:

    radius_server:

    nas_assignments:

    trusted_radius_client_assignments:
```

Each Tenant references:

- exactly one Authentication Profile

Each Tenant owns:

- exactly one Database
- exactly one RADIUS Server
- zero or more NAS Assignments
- zero or more Trusted RADIUS Client Assignments

Each tenant represents an independent FreeRADIUS deployment.

Each tenant ultimately produces an independent managed FreeRADIUS configuration tree.

A tenant without any NAS Assignments is considered incomplete and is not valid.

## RADIUS Server

Each Tenant contains one `radius_server` object.

```yaml
radius_server:
  version: 3.2.10
  authentication_port: 1812
  accounting_port: 1813
  coa_port: 3799
```

The `version` field specifies the target FreeRADIUS version for the deployment.

The configured version determines:

- which managed template set is used
- which deployment implementation is selected
- which version-specific validation rules apply

The port properties define the desired listener ports for the deployed FreeRADIUS instance.

The deployment layer is responsible for exposing those ports in the target runtime environment.

---

# Object Collections

Every object collection uses the same pattern.

The YAML key is the object's identifier.

Example:

```yaml
credential_profiles:

  default:
    shared_secret: secret1

  backup:
    shared_secret: secret2
```

The identifiers (`default` and `backup`) become the object identifiers referenced throughout the configuration.

Object identifiers must be unique within their respective collection.

---

# Relationships

Relationship objects reference other objects by identifier.

For example:

```yaml
nas_assignments:

  mt-core-01.gobcn.ca:

    credential_profile: default

    accounting_profile: default

    monitoring_profile: default

    require_message_authenticator: yes

trusted_radius_client_assignments:

  sonar:

    credential_profile: default

    require_message_authenticator: no
```

The key of an NAS Assignment is the identifier of its referenced NAS Device. The key of a Trusted RADIUS Client Assignment is the identifier of its referenced Trusted RADIUS Client. This keeps the assignment tenant-scoped while avoiding a redundant reference property.

`require_message_authenticator` is optional on either assignment type. When omitted, RADIUS Director does not generate the corresponding FreeRADIUS directive. When specified, it must be `auto`, `yes`, or `no`, and RADIUS Director generates that value.

Relationship objects never duplicate configuration owned by other objects.

Referenced objects must exist within the configuration.

An Accounting Profile's accounting policy applies to each NAS Assignment that references it.

## Migrating Assignment Identifiers

NAS Assignments and Trusted RADIUS Client Assignments previously used a separate assignment identifier and reference property. The assignment key now directly identifies the referenced Global Object.

Existing configurations using the previous assignment structure must be updated.

### NAS Assignments

Previously, an NAS Assignment used its own identifier and referenced an NAS Device using `nas_device`:

```yaml
nas_assignments:

  core-router:

    nas_device: mt-core-01.gobcn.ca

    credential_profile: default

    accounting_profile: default

    monitoring_profile: default
```

Move the `nas_device` value to the assignment key and remove the `nas_device` property:

```yaml
nas_assignments:

  mt-core-01.gobcn.ca:

    credential_profile: default

    accounting_profile: default

    monitoring_profile: default
```

The assignment key `mt-core-01.gobcn.ca` now directly identifies the Global NAS Device.

### Trusted RADIUS Client Assignments

Previously, a Trusted RADIUS Client Assignment used its own identifier and referenced a Trusted RADIUS Client using `trusted_radius_client`:

```yaml
trusted_radius_client_assignments:

  billing-system:

    trusted_radius_client: sonar

    credential_profile: default
```

Move the `trusted_radius_client` value to the assignment key and remove the `trusted_radius_client` property:

```yaml
trusted_radius_client_assignments:

  sonar:

    credential_profile: default
```

The assignment key `sonar` now directly identifies the Global Trusted RADIUS Client.

Obsolete `nas_device` and `trusted_radius_client` assignment properties should be removed when migrating existing configurations. YAML decoding is currently non-strict, so obsolete properties may otherwise be ignored rather than reported directly.

---

# Trusted RADIUS Clients

Trusted RADIUS Clients represent systems that communicate with FreeRADIUS but are not NAS Devices.

Examples include:

- billing systems
- provisioning systems
- monitoring systems
- management platforms
- other RADIUS servers

Trusted RADIUS Clients are reusable Global Objects.

They participate in client authentication but are not used when generating CoA proxy configuration.

Trusted RADIUS Client Assignments associate Trusted RADIUS Clients with Credential Profiles for a particular tenant.

Trusted RADIUS Clients do not reference Accounting Profiles and do not participate in stale-session accounting policy.

---

# Schema Evolution

Configuration compatibility should be versioned.

Breaking changes should include migration guidance.

---

# Validation

Configuration validation should detect:

- invalid YAML structure
- unsupported properties
- missing required properties
- invalid property types
- missing references
- duplicate identifiers
- invalid IP addresses
- invalid durations
- non-positive duration values
- invalid object relationships
- unsupported object combinations
- schema version mismatches

Configuration validation should report as many independent errors as practical during a single execution.

No managed configuration should be generated if validation fails.
