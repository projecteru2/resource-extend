# resource-extend

Out-of-tree resource plugins for [eru core](https://github.com/projecteru2/core): `resource-gpu` schedules
GPUs by product model and `resource-storage` schedules volumes, disk IOPS/BPS quota and storage space. Both
ship as standalone binaries that core executes per request — a subcommand per plugin verb, JSON on stdin,
JSON on stdout — so a cluster gains either resource without rebuilding core.

**Documentation: [projecteru2.github.io/resource-extend](https://projecteru2.github.io/resource-extend/)** (source in [`docs/`](docs/))

## Highlights

- **GPU scheduling by product** — a node declares how many cards of each model it has (`nvidia-3090: 4`) and
  a workload asks for a per-model count; allocation, realloc deltas and per-model metrics follow.
- **Volume, disk and storage scheduling** — `AUTO` volume bindings are placed on real devices, monopoly (`m`)
  bindings take a device to themselves, and read/write IOPS and BPS quota is tracked per disk.
- **One binary contract** — both plugins implement core's `resource/plugins.Plugin` and share one command
  tree, so a new plugin verb lands in both at once.
- **etcd-backed** — per-node capacity and usage live under `/resource/gpu/<node>` and `/resource/storage/<node>`
  in the same etcd cluster core uses.

## Quick start

```bash
make build                       # produces ./resource-gpu and ./resource-storage
cp resource-gpu resource-storage /etc/eru/plugins/

cp gpu.yaml.sample     /etc/eru/plugins/gpu.yaml
cp storage.yaml.sample /etc/eru/plugins/storage.yaml
```

Point core at the directory and it picks both up on the next start:

```yaml
# /etc/eru/core.yaml
resource_plugin:
  dir: /etc/eru/plugins
  call_timeout: 30s
```

Each plugin reads its own yaml from the working directory core runs it in, or from
`ERU_RESOURCE_CONFIG_PATH`. See [installation](docs/installation.md) and
[configuration](docs/configuration.md).

## Related projects

- [core](https://github.com/projecteru2/core) — the scheduler that calls these plugins
- [agent](https://github.com/projecteru2/agent) — per-node agent
- [cli](https://github.com/projecteru2/cli) — command line client, source of the node and workload resource options

## Development

```bash
make build       # build both plugin binaries
make test        # go vet on linux and darwin, then go test -race
make lint        # golangci-lint on linux and darwin
make fmt         # gofumpt + goimports
make help        # every target
```

## License

MIT, see [LICENSE](LICENSE).
