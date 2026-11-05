# Grantline

**See what changes before replacing an Atlas API key with a service account.**

[Try the browser workbench](https://balkashynov.github.io/grantline/) · [Validation](EVIDENCE.md) · [Design and input contract](DESIGN.md)

During the overlap phase of an Atlas access migration, both identities exist. Grantline compares their **exact scoped role assignments and allowed address sets**, then produces a reviewable diff with concrete IP witnesses.

For example, a replacement missing one project role and half a `/24` produces:

```text
Removed assignment: GROUP_DATA_ACCESS_READ_ONLY on project 2222…2222
Removed addresses:  192.0.2.128/25 (128 addresses)
Witness:            192.0.2.128
```

Two `/25` entries replacing one `/24` produce no network difference. An empty list is interpreted using the explicit organization IP-list requirement; unknown policy produces an incomplete result.

## Try it

The [browser workbench](https://balkashynov.github.io/grantline/) runs the same Go engine as the CLI, compiled to WebAssembly in a worker. Edit the examples, compare, and download a JSON report. Inputs stay in the tab; there is no backend or analytics. Examples are synthetic, not customer data or a live Atlas connection.

Local use requires Go 1.26+ and no cloud credentials:

```sh
go build -o grantline ./cmd/grantline
./grantline compare -input examples/lost-access.json
```

Exit codes: `0` = equal within scope, `2` = differences, `3` = incomplete, `1` = invalid input. A difference is an expected result, not a crash.

## Use Terraform state

Follow [MongoDB's migration guide](https://github.com/mongodb/terraform-provider-mongodbatlas/blob/v2.21.0/docs/guides/migrate-pak-to-service-account.md) through the overlap phase, when the old key and replacement service account are both present. Then:

```sh
# State can contain secrets. Keep this file local and out of Git.
terraform show -json > state.json

./grantline import \
  -state state.json \
  -from mongodbatlas_api_key.legacy \
  -to mongodbatlas_service_account.replacement \
  -require-ip-list true > comparison.json

./grantline compare -input comparison.json
```

Set `-require-ip-list` to the **actual organization setting** (`true`, `false`, or `unknown`). This is the Atlas Administration API access list, not a cluster's database network access list. The importer does not infer it from unrelated Terraform resources.

The importer selects only organization ID, resource-address labels, scoped roles and IP/CIDR entries. It omits secrets, public keys, client IDs, unrelated outputs and credentials from its output. Resource labels, organization/project IDs, roles and IPs still describe infrastructure: review them before sharing. Use normalized `comparison.json` in the browser, not raw state.

Supported resources, including nested modules:

| Purpose | API key | Service account |
|---|---|---|
| Organization identity | `mongodbatlas_api_key` | `mongodbatlas_service_account` |
| Project role assignment | `mongodbatlas_api_key_project_assignment` | `mongodbatlas_service_account_project_assignment` |
| Organization API access list | `mongodbatlas_access_list_api_key` | `mongodbatlas_service_account_access_list_entry` |

Import accepts `terraform show -json` **state format 1.x**, resource schema version 0, from `mongodb/mongodbatlas`. Plans, raw `.tfstate`, project-created identity resources, unresolved IDs, and ambiguous network fields fail with an error. Read [the complete contract](DESIGN.md) before integrating it into CI.

## What this adds

Terraform's normal plan checks declared resources against their state. MongoDB's guide explains how to create the replacement, copy assignments and test it. Grantline adds a cross-identity comparison: scope-sensitive role differences, CIDR union equivalence, exact missing/added ranges, and the empty-list policy distinction. It complements those tools; it does not perform the migration.

The core uses sorted disjoint intervals and arbitrary-precision integer endpoints. It subtracts sets without enumerating addresses, then emits canonical CIDRs and exact address counts. Role comparisons are set differences over `(scope, role)` pairs.

## Boundaries

- Matching assignments are **not a proof of effective permissions**. Role inheritance, custom authorization behavior, and API operation support are not modeled.
- Only supplied or Terraform-managed configuration is compared. Resources outside that state and live drift are not discovered.
- Credential validity, OAuth token issuance, rotation timing and application behavior are not tested.
- IPv6 is supported by the set engine; this project has not verified which IPv6 entries Atlas accepts.
- No Atlas account was used for validation. The provider integration uses real Terraform and provider schemas with **mocked** resource creation.
- A match is not permission to delete the old key. Reconcile unmanaged resources and run real application checks first.

## Reproduce validation

```sh
go test ./...
mkdir -p bin
go build -o bin/grantline ./cmd/grantline
./scripts/build-web.sh
node scripts/test-wasm.cjs
```

The optional provider integration requires Terraform 1.13.5 and downloads the pinned provider 2.21.0. It creates no cloud resources and needs no Atlas credentials:

```sh
terraform -chdir=integration/terraform init -input=false
terraform -chdir=integration/terraform test -json -verbose > integration/terraform/test-events.jsonl
python3 scripts/check-integration.py
```

See [EVIDENCE.md](EVIDENCE.md) for exact checks, retained reports and verification limits.

Independent open-source project by Ivan Balkashynov. Not affiliated with or endorsed by MongoDB.
