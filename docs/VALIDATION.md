# Validation

This document defines the validation rules applied to the RADIUS Director configuration model.

Validation ensures that the configuration is internally consistent before any FreeRADIUS configuration is generated.

Configuration generation **must not** proceed if validation fails.

---

# Validation Goals

Validation should:

- detect configuration errors as early as possible
- produce clear and actionable error messages
- report as many independent errors as practical in a single run
- prevent generation of invalid FreeRADIUS configuration
- validate the configuration model rather than generated files

---

# Validation Philosophy

Validation should be strict.

Configuration that cannot be generated correctly must be rejected.

The validator should:

- continue validation where practical
- report as many independent errors as possible during a single execution
- avoid stopping after the first error
- avoid reporting cascading errors that are only a consequence of an earlier failure

For example:

- A missing shared secret and an invalid NAS IP address should both be reported.
- An invalid database configuration should not prevent unrelated NAS Devices from being validated.
- A missing referenced object should be reported once as a reference validation error. The validator should not attempt to validate an object that does not exist.

The validator should never generate configuration that is known to be invalid.

---

# Validation Stages

Validation is performed in several stages.

## 1. Schema Validation

Verifies that the configuration is syntactically valid.

Examples include:

- required properties
- supported property names
- property types
- valid YAML structure

---

## 2. Object Validation

Each object validates only its own properties.

Object validation should not depend on other objects.

Examples include:

- required fields
- valid IP addresses
- supported database engines
- supported vendors

---

## 3. Reference Validation

Ensures that all object references exist.

Reference validation verifies relationships between objects but does not validate the referenced object's internal state.

Examples include:

- NAS Assignment references an existing NAS Device
- NAS Assignment references an existing Credential Profile
- Tenant references an existing Authentication Profile
- NAS Assignment references an existing Accounting Profile
- NAS Assignment references an existing Monitoring Profile

---

## 4. Relationship Validation

Ensures that relationships across the configuration are valid.

Unlike Object Validation, these rules consider multiple objects together.

Examples include:

- duplicate IP addresses
- duplicate NAS Assignments
- conflicting object relationships
- unsupported object combinations

---

## 5. Generation Validation

Verifies that sufficient information exists to generate FreeRADIUS configuration.

Examples include:

- every tenant has a database
- every tenant has exactly one RADIUS Server
- every tenant has at least one NAS Assignment

---

# Object Validation Rules

## Credential Profile

Validation rules:

- identifier must be unique
- shared_secret must be specified

---

## Authentication Profile

Validation rules:

- identifier must be unique

Additional validation is implementation specific.

---

## Accounting Profile

Validation rules:

- identifier must be unique

Additional validation is implementation specific.

---

## Monitoring Profile

Validation rules:

- identifier must be unique

Additional validation is implementation specific.

---

## NAS Device

Validation rules:

- identifier must be unique
- ip_address must be a valid IPv4 or IPv6 address
- ip_address must be unique
- vendor must be specified

---

## Tenant

Validation rules:

- authentication_profile must be specified
- referenced Authentication Profile must exist
- identifier must be unique
- exactly one Database must be defined
- exactly one RADIUS Server must be defined
- at least one NAS Assignment must be defined

---

## Database

Validation rules:

- engine must be supported
- host must be specified
- port must be valid
- database must be specified
- username must be specified
- password must be specified

Supported database engines:

- mysql

Additional database engines may be added in future releases.

---

## RADIUS Server

Validation rules:

- version must be specified
- version must be supported
- authentication_port must be between 1 and 65535
- accounting_port must be between 1 and 65535
- coa_port must be between 1 and 65535

A FreeRADIUS version is supported when the configured template library contains
at least one managed template set for that version. The default template-library
location is `./templates`; deployment YAML selects template-set and overlay
identifiers, not filesystem paths.

---

## NAS Assignment

Validation rules:

- identifier must be unique
- credential_profile must be specified
- accounting_profile must be specified
- monitoring_profile must be specified
- require_message_authenticator, when specified, must be `auto`, `yes`, or `no`
- referenced NAS Device must exist
- referenced Credential Profile must exist
- referenced Accounting Profile must exist
- referenced Monitoring Profile must exist

The NAS Assignment identifier is the referenced NAS Device identifier.

---

## Trusted RADIUS Client Assignment

Validation rules:

- identifier must be unique
- credential_profile must be specified
- require_message_authenticator, when specified, must be `auto`, `yes`, or `no`
- referenced Trusted RADIUS Client must exist
- referenced Credential Profile must exist

The Trusted RADIUS Client Assignment identifier is the referenced Trusted RADIUS Client identifier.

---

# Validation Errors

Validation errors should:

- identify the object containing the error
- identify the property causing the error
- describe the problem clearly
- provide enough information for the user to locate and correct the problem

Example:

```
tenant "residential":
nas assignment "mt-core-01.gobcn.ca":
credential profile "default" does not exist
```

Error messages should follow normal Go conventions:

- begin with a lowercase letter
- avoid trailing punctuation
- include sufficient context to uniquely identify the object being validated
