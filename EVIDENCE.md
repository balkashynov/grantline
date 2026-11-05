# Validation evidence

Local validation performed November 5, 2026 on macOS arm64 with Go 1.26.2, Node 24.3.0, Terraform 1.13.5, and MongoDB Atlas provider 2.21.0. All network and identity examples are synthetic.

## Central claim

The network engine computes exact set differences, rather than comparing CIDR strings. A seeded independent test generates 3,000 pairs of random prefix lists in `192.0.2.0/24`, checks all 256 addresses with `netip.Prefix.Contains`, and compares membership in A-minus-B against the engine output: **768,000 membership checks**, plus witness checks. The oracle enumerates addresses; the production algorithm uses interval subtraction.

Boundary cases cover a `/24` split into two `/25`s, a single missing last IPv4 address, full IPv6 space split into halves, the final IPv6 address, mixed families, exact `/0` cardinalities, and mapped IPv6 rejection. These tests are evidence, not a proof over all inputs.

## Importer and end-to-end checks

- Exact scoped role deltas, duplicate assignments, role changes without hierarchy assumptions, unknown organization policy, malformed schema and cross-organization rejection.
- Nested Terraform modules, unsupported format/schema versions, missing IDs/roles, malformed CIDRs, equivalent and conflicting computed IP/CIDR fields.
- A conspicuous synthetic secret sentinel is present in fixture state and is absent from comparison/report output and error messages for tested malformed values.
- Six examples execute through both the compiled CLI and the actual Go WebAssembly engine under Node. Complete JSON reports match, including hashes and exact counts.
- Real Terraform 1.13.5 loads the signed, pinned MongoDB provider 2.21.0 and executes `terraform test` with a mock provider. Its exported `test_state` values are wrapped in the documented state envelope, imported without modifying resource values, and compared. The expected missing role and `192.0.2.128/25` range are asserted.

Retained outputs:

- [Provider-mock normalized input](evidence/terraform-mock-comparison.json)
- [Provider-mock comparison report](evidence/terraform-mock-report.json)
- [Machine-readable local verification](evidence/verification.json)

Reproduction commands are in the README. `scripts/check-integration.py` fails unless the mock run passed and the expected deltas and sentinel omission hold. The Terraform lockfile pins provider hashes. Core Go code uses only the standard library.

## What was not verified

No live Atlas account, customer state, actual authentication cutover or production permission evaluation was exercised. Provider mocking validates resource schemas and Terraform plumbing; it does not execute Atlas API behavior. The IP-domain model follows public documentation, with explicit limits in the design.

Local browser automation is disabled in the available Work settings. The Go/WASM runtime, JavaScript syntax and static structure were checked; visual layout, DOM interactions and browser download behavior have not been exercised in a browser. This limit is retained rather than presenting a Node engine check as browser QA.
