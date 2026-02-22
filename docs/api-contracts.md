# API Contracts (Day 1 / Step 4)

This step defines wire contracts only. No service behavior is implemented yet.

## MetadataService

- `Create`: create file or directory metadata entries.
- `Lookup`: resolve a child entry by `(parent_inode_id, name)`.
- `Stat`: return metadata for one inode.
- `ListDir`: list entries under a directory inode.
- `Unlink`: remove one child entry from a parent.

`StripeLayout` is included in inode metadata so file placement is explicit from day one.

## ObjectStorageService

- `WriteBlock`: write one block for a file/chunk/OST tuple.
- `ReadBlock`: read block bytes with offset and length.
- `DeleteBlock`: remove one block.
- `GetHealth`: return basic node health and throughput/IOPS counters.

`BlockRef(file_id, chunk_id, ost_id)` is the stable identifier across MDS and OST calls.

## Generation and verification commands

- Generate stubs: `make proto-gen`
- Compile check: `make compile-check`

## What to observe in Step 4

- Generation is deterministic into `pkg/proto/gen`.
- Build should fail early if `protoc-gen-go` or `protoc-gen-go-grpc` is missing.
- Compile check should surface toolchain issues before application logic is added.
