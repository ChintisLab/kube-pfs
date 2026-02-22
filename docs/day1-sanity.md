# Day 1 Sanity Checks

Run these checks before implementing application logic.

## Commands

- `make sanity-tooling`
- `make sanity-proto-repro`
- `make sanity-container-smoke`
- `make sanity` (runs all three)

## What each check validates

1. Tooling check
- Ensures required local tools exist and satisfy supported version constraints.
- Fails fast with a clear message when a dependency is missing.
- Verifies Docker daemon connectivity before cluster/image tasks.

2. Proto reproducibility check
- Runs protobuf generation twice.
- Hashes generated files and confirms outputs are identical.
- Fails if generated files are unstable across runs.

3. Container smoke build check
- Builds smoke images for `mds`, `ost`, `csi-controller`, and `csi-node`.
- Validates Docker build context and image pipeline wiring.

## What to observe

- If tooling fails, do not continue to implementation.
- If proto reproducibility fails, fix generation determinism before coding services.
- If smoke builds fail, fix Docker baseline before adding runtime images.
