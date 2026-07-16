# Compatibility release gate

`stack-lock.json` is a release manifest, not a claim that compatibility has
already been proven. Keep its status as `candidate` while evidence is being
collected.

A release is permitted only after every name in `requiredGates` has a matching
`gateResults` entry with:

- `status`: `passed`
- `evidence`: a JSON path below `compatibility/evidence/`
- `sha256`: the lowercase SHA-256 digest of that committed JSON file

Each evidence document must use schema `1`, name the gate, record status
`passed`, pin `testedCommit` to the lock's `idena-go.runtimeCodeCommit`, and
include an HTTPS `source` URL for the underlying logs or attestation. The
release check recomputes the digest and validates these fields locally; it does
not trust a digest-shaped string by itself.

The `independent-rebuild-digest-match` evidence must also record the exact
`releaseTag` and a `releaseArtifacts` entry for each supported platform. Every
entry contains `platform` and `sha256`; the release job hashes the newly built
binaries and refuses to publish them unless all five digests match this
committed evidence. Release builds disable VCS stamping so metadata-only commits
after `runtimeCodeCommit` do not invalidate independently reproduced binaries.

After independently reviewing all evidence, change the top-level status to
`approved`. The release workflow rejects candidate locks, incomplete evidence,
tags outside the default branch, missing artifacts, and checksum mismatches.

Do not approve a lock based only on unit tests. The block/RPC differential,
state replay, mixed-node P2P, Wasm receipt and gas, cross-architecture,
reproducible build, secret scan, and dependency gates are separate controls.
