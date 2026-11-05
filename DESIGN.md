# Design and input contract

## Scope

Grantline answers: which exact scoped role assignments and address sets differ between two supplied identities in one Atlas organization? It does not answer whether either identity can execute a particular API operation.

The intended user is an engineer reviewing the overlap phase of a Terraform-managed API-key to service-account migration. The result supports a concrete decision: which copied assignments or network entries need correction or explicit review before application tests and key retirement.

## Architecture

```text
Terraform state JSON -> whitelisted importer -> comparison JSON -> Go engine -> report JSON
                                                   ^                 |
                                                   |                 |
                                       browser editor -------- Go/WASM worker
```

The browser runs the engine, not recorded results. It loads static code and examples; it sends no comparison inputs to a server. No service backend, account, local storage or telemetry is used. Reports include a SHA-256 of the canonical Go JSON encoding of the input, binding each result to its exact supplied snapshot. This is not an attestation or semantic hash; different ordering or redundant entries can produce different hashes.

## Comparison input

See the complete files in `examples/`. Required fields:

- `schemaVersion`: 1.
- `scope`: `provided-snapshot` or `terraform-managed`; neither means complete live Atlas inventory.
- `before` and `after`: label, 24-hex organization ID, explicit `grants` and `accessList` arrays, and `requireIpList` boolean or null.
- Each grant is `{ "scope": "org:ID" | "project:ID", "role": "ORG_…" | "GROUP_…" }`. Organization-scoped IDs must match the identity's organization.

Unknown top-level or identity fields are rejected. Labels are limited to 160 characters; lists to 10,000 entries each; input to 16 MiB. Error messages do not echo malformed field values. These are resource bounds, not a hardened adversarial service boundary.

Role names are opaque, scope-checked labels. New role labels can be compared without inventing a role hierarchy. An owner replacing a member is a removed assignment plus an added assignment; the engine does not assert their permission relationship.

## Address-set algebra

Parse IPs or prefixes with Go `net/netip`; normalize host bits and collapse duplicate, overlapping and adjacent ranges. IPv4 and IPv6 remain distinct families. IPv4-mapped IPv6 and zone IDs are rejected rather than silently reinterpreted.

Each prefix becomes an inclusive integer interval. `big.Int` handles the full 128-bit domain, including the last address and a temporary sentinel beyond the maximum during subtraction. Sorting and union cost O(n log n). Difference uses a two-pointer sweep over normalized sets, then decomposes each changed interval into aligned power-of-two blocks. Output is deterministic, ordered and nonoverlapping; each block's first address is a witness. No address space is enumerated by the production algorithm.

Empty access lists follow the documented Atlas Administration API organization setting:

- Required = true: empty set.
- Required = false: unrestricted address domain in the model.
- Unknown: no network verdict, overall status `incomplete`, while exact role differences remain visible.

The unrestricted model contains IPv4 and IPv6. That is a mathematical domain, not evidence that Atlas will accept every address or permit every operation. API method restrictions, token endpoints and runtime conditions are outside this comparison.

## Terraform importer

The importer accepts the state representation produced by `terraform show -json`, not the raw state storage format or a plan. It recursively visits root and child modules and selects the exact two managed resource addresses. IDs associate project assignments and access-list entries with those principals, even when resources live in separate modules.

It supports the six organization-level resource types in the README, provider name `registry.terraform.io/mongodb/mongodbatlas`, resource schema version 0. Unknown values, missing references, duplicate managed addresses, cross-organization comparisons and unsupported identity/access resources referencing the selected principal are rejected. Secret lifecycle resources are ignored explicitly because their behavior is outside the result.

Both `cidr_block` and `ip_address` are optional/computed in the provider. One populated field is normal. When both exist, only equivalent singleton forms (for example `192.0.2.10/32` and `192.0.2.10`) are accepted; ambiguous dual fields are rejected. The importer does not choose silently between contradictory representations.

Data sources are not treated as managed resources. Missing unmanaged assignments cannot be discovered from this file. The snapshot can be stale, and grants from other organizational mechanisms may affect actual access. Consequently the strongest status is `equal_in_scope`, never `safe` or `authorized`.

## Status and export

`differences_found` means at least one scoped assignment or network range differs. `incomplete` takes precedence when organization policy is unknown. Invalid inputs return an error and no report. `equal_in_scope` means exact compared sets match, subject to all stated limits.

The UI caps displayed findings at 12 per category to remain usable. The downloadable report contains every finding. New edits invalidate the previous result. Long comparisons run in a worker with a UI timeout; reload or the CLI is available if the worker stops.

## Sources and existing work

Reviewed November 5, 2026:

- [MongoDB provider migration guide, pinned v2.21.0](https://github.com/mongodb/terraform-provider-mongodbatlas/blob/v2.21.0/docs/guides/migrate-pak-to-service-account.md): overlap phase and resource mapping. Migration is recommended, not mandatory.
- [Atlas Administration API setup](https://www.mongodb.com/docs/atlas/configure-api-access/): organization IP policy and empty-list behavior. Access lists govern API requests using tokens, not token creation.
- [Authentication methods](https://www.mongodb.com/docs/atlas/api/api-authentication/): service accounts and legacy API keys; Atlas control-plane access differs from database credentials.
- [Terraform JSON format](https://developer.hashicorp.com/terraform/internals/json-format): state vs plan and version compatibility.
- [Provider IP-key implementation](https://github.com/mongodb/terraform-provider-mongodbatlas/blob/v2.21.0/internal/service/accesslistapikey/resource.go) and [service-account model](https://github.com/mongodb/terraform-provider-mongodbatlas/blob/v2.21.0/internal/service/serviceaccountaccesslistentry/model.go): computed IP/CIDR fields.

A textual list diff is sufficient for small, already-normalized configurations. The official guide is the right tool for carrying out the migration. Grantline is useful when copied assignments span projects/modules or network rewrites make a textual diff hard to interpret. No adopter has requested or validated this exact workflow yet.
