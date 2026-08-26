# Configuration

Both plugins read one yaml file. It is a subset of core's own config, parsed with core's loader, so the key
names and defaults match core exactly. Only the keys below are used.

## gpu.yaml

```yaml
etcd:
    machines:
        - http://127.0.0.1:2379
    prefix: "/eru-gpu"
```

## storage.yaml

```yaml
etcd:
    machines:
        - http://127.0.0.1:2379
    prefix: "/eru-storage"

scheduler:
    max_deploy_count: 50
```

## Keys

| Key | Used by | Required | Default | Meaning |
|---|---|---|---|---|
| `etcd.machines` | both | yes | — | etcd endpoints holding per-node capacity and usage |
| `etcd.prefix` | both | yes | `/eru` | key namespace; the plugin writes `<prefix>/resource/gpu/<node>` or `<prefix>/resource/storage/<node>` |
| `etcd.lock_prefix` | both | no | `__lock__/eru` | inherited from core's config struct, unused by the plugins |
| `etcd.ca`, `etcd.cert`, `etcd.key` | both | no | — | TLS material for the etcd client |
| `etcd.auth.username`, `etcd.auth.password` | both | no | — | etcd authentication |
| `scheduler.max_deploy_count` | storage | no | `10000` | upper bound on the deploy capacity one node reports and on the volume plans returned for it |

`scheduler.max_deploy_count` bounds placement, not work. It is the deploy capacity a node reports when a
request needs no volume placement, and otherwise the cap on the plans `get-nodes-deploy-capacity` and
`calculate-deploy` return per node — the enumeration behind them runs until the node is full regardless of
the cap. Raising it lets a node report a larger deploy capacity for small volume requests.

The GPU plugin has no scheduler section — GPU capacity is an integer count per product, so a node's capacity
is computed directly rather than enumerated.

## Config path

A plugin resolves its config in this order:

1. `--config <path>` on the command line, before the subcommand.
2. `ERU_RESOURCE_CONFIG_PATH` in the environment.
3. `gpu.yaml` / `storage.yaml`, relative to the working directory.

Core sets the working directory to `resource_plugin.dir` before executing a plugin, so option 3 resolves
next to the binary. Core does not set `ERU_RESOURCE_CONFIG_PATH` itself; export it in core's own
environment to point both plugins elsewhere.

A missing or incomplete config is a hard failure: the plugin exits non-zero with
`Machines is required, but blank` rather than starting without a store.
