# Domain Model

This document defines the core objects managed by RADIUS Director.

The domain model intentionally describes operational concepts rather than FreeRADIUS configuration files.

Managed FreeRADIUS configuration is derived from this model.

---

# Global Objects

Global Objects are reusable objects that may be referenced by one or more tenants.

Global Objects include:

- Credential Profiles
- Authentication Profiles
- Accounting Profiles
- Monitoring Profiles
- NAS Devices
- Trusted RADIUS Clients

---

# Tenant Objects

Each tenant contains tenant-specific infrastructure and references a tenant-wide Authentication Profile.

Tenant Objects include:

- Database
- RADIUS Server

Relationship Objects owned by a tenant define how Global Objects are composed into the tenant's deployment.

Each tenant represents an independent FreeRADIUS deployment.

Each tenant ultimately produces its own managed FreeRADIUS configuration tree.

Tenants reference Global Objects rather than duplicating them.

---

# Relationship Objects

Some objects exist primarily to describe relationships between other objects.

Relationship Objects allow reusable Global Objects to be composed into tenant-specific deployments.

Relationship Objects include:

- NAS Assignment
- Trusted RADIUS Client Assignment

A NAS Assignment references:

- NAS Device
- Credential Profile
- Accounting Profile
- Monitoring Profile

A Trusted RADIUS Client Assignment references:

- Trusted RADIUS Client
- Credential Profile

Relationship Objects remain owned by a single tenant.

---

# Credential Profile

Defines reusable authentication credentials.

Typical properties include:

- RADIUS shared secret
- CoA shared secret

Multiple NAS Assignments and Trusted RADIUS Client Assignments may reference the same Credential Profile.

---

# Authentication Profile

Defines tenant-wide subscriber authentication policy.

Typical properties include:

- Simultaneous Use

Each tenant references one Authentication Profile. Authentication Profiles are reusable Global Objects and may be shared by multiple tenants.

---

# Accounting Profile

Defines accounting behaviour.

Examples include:

- Accounting storage
- Interim update handling
- Stale session cleanup
- Retention policies

Accounting Profiles are referenced by NAS Assignments.

An Accounting Profile may define when an active accounting session should be considered stale.

Stale session detection is based on the absence of sufficiently recent accounting activity. This allows sessions for which an Accounting-Stop packet was never received to be closed automatically after a configurable timeout.

When a stale session is closed, the recorded stop time should represent the last known accounting activity for the session rather than the time at which the cleanup process happened.

Stale session cleanup is independent of session verification performed during authentication. Session verification may determine that a session is no longer active and allow a subscriber to reconnect, while stale session cleanup is responsible for maintaining accurate accounting records.

The Accounting Profile defines accounting policy, including the stale session timeout. The mechanism and schedule used to execute periodic accounting maintenance are operational concerns and are not part of the Accounting Profile.

---

# Monitoring Profile

Defines operational monitoring.

Examples include:

- Ping tests
- Authentication tests
- CoA tests
- SNMP verification

Monitoring Profiles are referenced by NAS Assignments.

---

# NAS Device

Represents a physical or virtual RADIUS client.

A NAS Device is a reusable Global Object that describes the device itself, independent of how it is used by any particular tenant.

Typical properties include:

- Name
- Address
- Vendor
- Model
- Description
- Tags

Operational behaviour such as credentials, authentication, accounting, and monitoring is defined through NAS Assignments rather than directly on the NAS Device.

---

# Trusted RADIUS Client

Represents a trusted system that communicates with FreeRADIUS but is not itself a NAS Device.

Examples include:

- billing systems
- provisioning systems
- monitoring systems
- management platforms
- other RADIUS servers

Trusted RADIUS Clients are reusable Global Objects.

Unlike NAS Devices, Trusted RADIUS Clients participate only in client authentication and authorization.

They are not used when generating CoA proxy configuration.

Typical properties include:

- Name
- Address
- Description

Operational behaviour is defined through Trusted RADIUS Client Assignments.

---

# NAS Assignment

Represents how a tenant uses a NAS Device.

A NAS Assignment is a Relationship Object that combines reusable Global Objects into tenant-specific managed configuration.

The NAS Assignment key is the identifier of its NAS Device. Each NAS Assignment references:

- Credential Profile
- Accounting Profile
- Monitoring Profile

It may optionally set `require_message_authenticator` to `auto`, `yes`, or `no`. When omitted, no corresponding FreeRADIUS directive is generated.

Multiple tenants may reference the same NAS Device while applying different operational policies through separate NAS Assignments.

---

# Trusted RADIUS Client Assignment

Represents how a tenant uses a Trusted RADIUS Client.

A Trusted RADIUS Client Assignment is a Relationship Object that combines a Trusted RADIUS Client with the credentials required for it to communicate with the tenant's FreeRADIUS deployment.

The Trusted RADIUS Client Assignment key is the identifier of its Trusted RADIUS Client. Each Trusted RADIUS Client Assignment references:

- Credential Profile

It may optionally set `require_message_authenticator` to `auto`, `yes`, or `no`. When omitted, no corresponding FreeRADIUS directive is generated.

Multiple tenants may reference the same Trusted RADIUS Client while using different Credential Profiles if required.

Trusted RADIUS Client Assignments do not participate in CoA proxy configuration.

---

# Database

Represents the primary database backing a tenant's RADIUS deployment.

Typical properties include:

- Database engine
- Host
- Port
- Database name
- Authentication credentials
- TLS configuration

Each tenant owns its own database definition.

The deployment layer determines how the configured database is connected to at runtime.

Future versions may support additional backend services where appropriate.

---

# RADIUS Server

Represents the desired runtime characteristics of a tenant's FreeRADIUS deployment.

A RADIUS Server defines how the tenant's managed configuration is rendered and how it is deployed.

Typical properties include:

- FreeRADIUS version
- Authentication port
- Accounting port
- CoA port

The configured FreeRADIUS version determines:

- which managed template set is used
- which version-specific validation rules apply
- which deployment implementation is selected

The port properties define the desired listener ports for the deployed FreeRADIUS instance.

The deployment layer is responsible for exposing those ports in the target runtime environment.
