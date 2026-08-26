# resource-extend

Out-of-tree resource plugins for [eru core](https://github.com/projecteru2/core). `resource-gpu` schedules
GPUs by product model; `resource-storage` schedules volumes, disk IOPS/BPS quota and storage space. Core
executes each plugin as a binary — one subcommand per verb, the request as JSON on stdin, the result as JSON
on stdout — so a cluster gains either resource without rebuilding core.

```
eru-core
   │  resource/cobalt
   │     ├─► cpumem            (built into core)
   │     └─► binary plugins    resource_plugin.dir
   │                │
   │                ├─ exec ./resource-gpu     <verb>  <── JSON ──> gpu.yaml
   │                └─ exec ./resource-storage <verb>  <── JSON ──> storage.yaml
   │                                  │
   └──────────────────────────────────┴──► etcd  /resource/{gpu,storage}/<node>
```

Both binaries come from one Go module:

```
plugincmd/   the shared urfave/cli command tree, one handler per plugin verb
nodestore/   per-node capacity and usage in etcd
gpu/         the GPU plugin       -> cmd/resource-gpu
storage/     the storage plugin   -> cmd/resource-storage
```

## Guides

- [Installation](installation.md) — building the binaries and wiring them into core
- [Configuration](configuration.md) — every key `gpu.yaml` and `storage.yaml` read
- [GPU plugin](gpu.md) — what it schedules, how capacity and usage are modelled, its metrics
- [Storage plugin](storage.md) — volumes, disks and storage space, and how a plan is built
- [Plugin protocol](protocol.md) — the binary contract with core: subcommands, JSON in and out

## Repository

Source and issue tracker: [github.com/projecteru2/resource-extend](https://github.com/projecteru2/resource-extend).
Part of the [Eru](https://github.com/projecteru2) cluster stack, alongside
[core](https://github.com/projecteru2/core), [agent](https://github.com/projecteru2/agent) and
[cli](https://github.com/projecteru2/cli).
