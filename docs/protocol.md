# Plugin protocol

Core talks to an out-of-tree resource plugin by executing its binary once per call. There is no daemon, no
socket and no persistent state in the process: the verb is `argv[1]`, the request is a JSON object on
stdin, and the response is a JSON object on stdout.

```
core: exec <plugin-binary> <verb>          cwd = resource_plugin.dir
      stdin  <- {"nodename": "node-1", ...}
      stdout -> {"capacity": {...}, "usage": {...}}
      exit 0
```

Core reads stdout as the response and stderr as the plugin's log, so stdout carries nothing but the JSON
result and an empty stdout is an error. On a non-zero exit core logs stderr with the failure. The call is
bounded by `resource_plugin.call_timeout`.

Core runs `verbs` once when it loads the plugin and never spawns it for a verb outside the answer; the
plugin takes no part in that call. Both plugins leave out `calculate-remap`, which they have nothing to
remap for.

The plugin name core uses for a resource is the binary's file name, not anything the plugin reports. The
`name` verb exists for humans.

## Verbs

| Verb | Request keys | Response |
|---|---|---|
| `name` | — | the plugin name, as a JSON string |
| `verbs` | — | array of the verbs this binary implements |
| `get-metrics-description` | — | array of `{name, help, type, labels}` |
| `get-metrics` | `nodes`: array of `{podname, nodename}` | array of `{name, labels, key, value}` for every node |
| `add-node` | `nodename`, `resource`, `info` | `{capacity, usage}` |
| `remove-node` | `nodename` | `{}` |
| `get-nodes-deploy-capacity` | `nodenames`, `workload_resource` | `{nodes_deploy_capacity_map, total}` |
| `set-node-resource-capacity` | `nodename`, `resource`, `resource_request`, `delta`, `incr` | `{before, after}` |
| `get-node-resource-info` | `nodename`, `workloads_resource` | `{capacity, usage, diffs}` |
| `set-node-resource-info` | `nodename`, `capacity`, `usage` | `{}` |
| `set-node-resource-usage` | `nodename`, `resource`, `resource_request`, `workloads_resource`, `delta`, `incr` | `{before, after}` |
| `get-most-idle-node` | `nodenames` | `{nodename, priority}` |
| `fix-node-resource` | `nodename`, `workloads_resource` | `{capacity, usage, diffs}` |
| `calculate-deploy` | `nodename`, `deploy_count`, `workload_resource_request` | `{engines_params, workloads_resource}` |
| `calculate-realloc` | `nodename`, `workload_resource`, `workload_resource_request` | `{engine_params, delta_resource, workload_resource}` |
| `calculate-remap` | `nodename`, `workloads_resource` | `{engine_params_map}` — not implemented by either plugin |

The verb names and the request shapes are core's, from `resource/plugins/binary`. This repository does not
define them; it implements them.

## Request and response values

- **`resource` / `resource_request`** — an object whose keys the plugin defines. `resource` is a parsed
  resource of the plugin's own shape; `resource_request` is what an operator typed, so it may hold strings
  where the resource holds numbers (`"1T"` against `1099511627776`).
- **`workloads_resource`** — an array of workload resources, one per workload on the node. Used to
  recompute usage and to report diffs.
- **`info`** — the engine's node report, marshalled from core's `enginetypes.Info`: engine type, node id,
  cpu count, memory and storage totals.
- **`delta` / `incr`** — `delta` false means the value replaces what is stored, true means it is applied to
  it; `incr` picks addition or subtraction. When `delta` is false the plugins force addition, because the
  absolute value is what was asked for.
- **`before`** — when a set fails in one plugin after succeeding in another, core rolls the survivors back
  by replaying each plugin's `before` with `delta` false, a capacity snapshot through `resource_request` and
  a usage snapshot through `resource`. A plugin must accept its own snapshot there: the GPU snapshot already
  has the request shape, and the storage plugin recognizes a request carrying `volumes`, `disks` and
  `storage` in their stored form and restores it verbatim.

## Locking

Every mutating verb is a read-modify-write on the node's record with no compare-and-swap. Core serializes
them by holding a node- or pod-scoped lock around every call that writes a plugin's usage or capacity, and
that lock is the whole guarantee: two writers on one node from outside core lose one update.

## Error handling

Any verb may fail. The plugin exits with status 128, and core surfaces the failure to the caller.

A node this plugin has never seen is not an error. `get-node-resource-info` returns `null` and exits 0, so
core reads an empty resource for that node instead of failing the whole `node get` or `node list`. The
node verbs that read several nodes (`get-nodes-deploy-capacity`, `get-most-idle-node`) treat such a node as
holding nothing of this kind: it stays deployable for requests that ask for nothing of this kind and has no
capacity for the others. `set-node-resource-capacity` and `set-node-resource-usage` start from that empty
record and create it, so `node set` is how an operator declares capacity on a node that predates the
plugin; `fix-node-resource` still needs the record. This matters for a resource added to a cluster after its nodes were: without it,
every node predating the plugin would break listing and deploying.

## Adding a verb

Both binaries share one command tree in `plugincmd/`. A verb is a `handler`:

```go
func setNodeResourceInfo(ctx context.Context, p plugins.Plugin, in resourcetypes.RawParams) (any, error) {
	node, err := nodename(in)
	if err != nil {
		return nil, err
	}
	return p.SetNodeResourceInfo(ctx, node, in.RawParams("capacity"), in.RawParams("usage"))
}
```

registered in `nodeCommands`, `calculateCommands` or `metricsCommands`. It is added to both plugins at
once, and it works for any type implementing core's `resource/plugins.Plugin`. `plugincmd/plugincmd_test.go`
pins the decoding against the request structs core actually sends.
