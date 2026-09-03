# GPU plugin

`resource-gpu` schedules whole GPU cards by product model. It does not divide a card, track GPU memory, or
pin a workload to a specific PCI address — a node has *n* cards of a model, a workload asks for *k* of that
model, and the plugin keeps the arithmetic straight across deploy, realloc and node edits.

## The resource

Both node and workload resources are one field:

```json
{"prod_count_map": {"nvidia-3090": 4, "nvidia-3070": 2}}
```

The key is a product model, free-form text chosen by whoever adds the node; the value is a card count. Blank
or whitespace-only products are rejected, and so are counts of zero or less — except in a realloc request,
where a negative count means "give these back".

A node record holds two of these maps:

- **capacity** — what the node has.
- **usage** — what its workloads hold.

Available capacity is capacity minus usage, per product. Both live in etcd under
`<etcd.prefix>/resource/gpu/<nodename>`.

## Adding a node

`add-node` takes the count map directly:

```bash
eru-cli node add --extra-resources '{"resource-gpu":{"prod_count_map":{"nvidia-3090":4}}}' ...
```

The request key `resource-gpu` is the plugin binary's file name — core names a plugin after the file it
executes, not after what the plugin's `name` subcommand reports.

A request that carries no cards registers a node with no GPU capacity; the engine's node report carries
no GPU information.

## Deploy capacity

For a request of *k* cards of a product, a node can host `available[product] / k` workloads; with several
products in one request the node capacity is the smallest of those quotients. A request that asks for no
GPUs at all leaves the node effectively unbounded — the plugin reports a capacity of 1000000, so GPU never
becomes the limiting resource for a workload that does not want a card.

`get-most-idle-node` picks the node with the lowest `usage / capacity` card ratio, the first by name on a tie, and reports priority 100,
which makes GPU the dominant voice when core weighs plugins for a build node.

## Realloc

A realloc request is merged into the workload's current cards before validation, so counts add up:

| Current | Request | Result |
|---|---|---|
| `{"nvidia-3090": 1}` | `{}` | `{"nvidia-3090": 1}` |
| `{"nvidia-3090": 1}` | `{"nvidia-3090": 2}` | `{"nvidia-3090": 3}` |
| `{"nvidia-3090": 1}` | `{"nvidia-3090": -1, "nvidia-3070": 1}` | `{"nvidia-3070": 1}` |

A negative count larger than what the workload holds clamps at zero rather than failing; the product simply
disappears from the map. The response also carries the delta, which core applies to node usage.

`calculate-remap` is a no-op: cards are not re-pinned between running workloads.

## Diffs and repair

`get-node-resource-info` compares stored node usage against the sum of the workload resources core passes
in, and reports every mismatch as a human-readable diff line — a total that disagrees, a product present in
the workloads but missing from usage, or a per-product count that differs. `fix-node-resource` runs the same
comparison and, when anything differs, overwrites node usage with the sum computed from the workloads; a
write that fails fails the verb, so core keeps its repair entry and replays it.

## Metrics

| Metric | Type | Labels | Value |
|---|---|---|---|
| `gpu_capacity` | gauge | `podname`, `nodename`, `product` | cards of that product on the node |
| `gpu_used` | gauge | `podname`, `nodename`, `product` | cards of that product held by workloads |

One pair is emitted per product in the node's capacity map. The statsd key is
`core.node.<nodename>.gpu.capacity` / `.gpu.used`, with dots in the node name replaced by underscores.
