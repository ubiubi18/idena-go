# Compatibility release gate

`stack-lock.json` is a release manifest, not a claim that compatibility has
already been proven. Keep its status as `candidate` while evidence is being
collected.

A release is permitted only after every name in `requiredGates` has a matching
`gateResults` entry with:

- `status`: `passed`
- `evidence`: an HTTPS URL to immutable logs or an attestation
- `sha256`: the lowercase SHA-256 digest of that evidence

After independently reviewing all evidence, change the top-level status to
`approved`. The release workflow rejects candidate locks, incomplete evidence,
tags outside the default branch, missing artifacts, and checksum mismatches.

Do not approve a lock based only on unit tests. The block/RPC differential,
state replay, mixed-node P2P, Wasm receipt and gas, cross-architecture,
reproducible build, secret scan, and dependency gates are separate controls.
