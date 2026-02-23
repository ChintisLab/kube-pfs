# Day 2 Development Notes

I implemented the Day 2 MVP in four parts:

1. MDS service
- In-memory inode map with BoltDB persistence.
- RPCs: `Create`, `Lookup`, `Stat`, `ListDir`, `Unlink`.
- Round-robin stripe ordering across configured OST IDs.

2. OST service
- Flat-file block storage keyed by `(file_id, chunk_id)`.
- RPCs: `WriteBlock`, `ReadBlock`, `DeleteBlock`, `GetHealth`.
- Basic IOPS and throughput counters.

3. CSI controller/node services
- Identity server implementation.
- Controller RPCs for create/delete/publish.
- Node RPCs for stage/publish/unstage/unpublish in local MVP mode.

4. Smoke validation
- A focused smoke test validates create/write/read/unlink path.
- `make smoke` runs this before larger integration work.

## What I can run now

```bash
make compile-check
make smoke
```

## What I will improve in Day 3+

- Replace local node publish marker behavior with full FUSE mount flow.
- Add Prometheus metrics endpoint and dashboards.
- Add fault injection jobs and benchmark report generation.
