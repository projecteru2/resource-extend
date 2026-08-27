# Storage plugin

`resource-storage` schedules three related things on a node:

- **volumes** — named devices with a byte capacity, e.g. `/data0: 1TiB`. A workload binding with source
  `AUTO` is placed on one of them by the scheduler.
- **disks** — physical devices with mount points and read/write IOPS and BPS quota. A binding that asks for
  IOPS draws that quota from whichever disk covers its path.
- **storage** — a flat byte budget for the node, covering both scheduled volumes and plain disk usage.

Capacity and usage live in etcd under `<etcd.prefix>/resource/storage/<nodename>`.

## Volume binding syntax

```
source:destination[:flags[:size[:read_IOPS:write_IOPS:read_BPS:write_BPS]]]
```

- `source` — a host path, or `AUTO` (or a path ending in `AUTO`, or empty) to have the plugin choose a device.
- `destination` — the path inside the workload. Required.
- `flags` — `r`, `w`, `o` and `m`, sorted; defaults to `rw`. `m` means monopoly.
- `size` — bytes, human sizes accepted (`100GiB`, `1T`).
- the four IOPS/BPS numbers — per-binding quota; sizes accept human units.

Four kinds of binding fall out of that:

| Kind | Condition | Placement |
|---|---|---|
| mount | source is a real path | not placed; only its IOPS quota is charged to the covering disk |
| normal | `AUTO` with a size | placed on a used device large enough, smallest first |
| monopoly | `AUTO` with flag `m` and a positive size | takes an unused device to itself; several monopoly bindings share one device proportionally |
| unlimited | `AUTO`, size 0 and no IOPS | placed on the device with the most free space, charged nothing |

Request and limit are separate lists (`volumes-request` and `volumes`). They must describe the same
bindings; where the limit is smaller than the request the plugin raises the limit to match, so a request is
never unschedulable against its own limit.

## Adding a node

```bash
eru-cli node add \
  --extra-resources '{"resource-storage":{"volumes":["/data0:1T","/data1:1T"],"disks":["/dev/vda:/,/data0:1000:1000:1G:1G"],"storage":"2T"}}' ...
```

The request key `resource-storage` is the plugin binary's file name — core names a plugin after the file it
executes, not after what the plugin's `name` subcommand reports.

| Key | Form | Meaning |
|---|---|---|
| `volumes` | `["device:size", ...]` | schedulable devices and their capacity |
| `disks` | `["device:mounts:read_IOPS:write_IOPS:read_BPS:write_BPS", ...]` | IOPS/BPS quota; `mounts` is comma separated |
| `storage` | `"80G"` or `85899345920` | flat storage budget; the total volume size is added to it |
| `rm-disks` | `"device,device"` | remove these disks; only valid in absolute mode, not with a delta |

Every storage size takes either form — the human string `"80G"` or the plain byte count `85899345920` — here
and in a workload's `storage`, `storage-request` and `storage-limit`.

When a node is added with no `storage` value, the plugin takes 80% of the engine's reported total storage.
In absolute mode (`delta` false) any key left out of the request keeps its stored value rather than being
reset to zero. An absolute request carrying `volumes`, `disks` and `storage` in their stored form — the
`before` snapshot of an earlier call — is recognized as core rolling back and restores the capacity
verbatim.

## How a plan is built

`get-nodes-deploy-capacity` and `calculate-deploy` both run the same scheduler:

1. Bindings are split into the four kinds above and sorted by size.
2. Normal bindings are packed onto used devices with a min-heap, smallest device that still fits first, so
   large free devices stay whole.
3. Monopoly bindings take unused devices; the requests on one device split it proportionally, with the
   rounding remainder going to the first of them.
4. The plugin then converts unused devices into used ones for as long as monopoly capacity exceeds normal
   capacity, and takes the plans from the last pass. A capacity query weighs the node's full potential;
   `calculate-deploy` opens unused devices no further than the requested count needs, keeping the rest
   whole for later monopoly requests.
5. Unlimited bindings are pinned to whichever device has the most space left after the other two passes.
6. Mount bindings never move; their IOPS quota is charged to the disk covering their source path.

Node deploy capacity is the number of plans that survive, capped by `scheduler.max_deploy_count`, and
further capped by `(capacity.storage - usage.storage) / storage-request` when the request asks for storage.
A request with no binding that needs placement or IOPS skips the scheduler entirely and reports
`scheduler.max_deploy_count`.

`get-most-idle-node` returns the first node it is given with priority 1: the storage plugin expresses no
preference about build placement and defers to the other plugins.

## Realloc

A realloc request is merged into the workload's current bindings — sizes and quotas add, so
`AUTO:/dir:rw:100GiB` against a workload already holding 100GiB yields 200GiB, and a negative size shrinks
it. Rescheduling only happens when the *request* touches a binding that needs placement or IOPS; otherwise
the existing volume and disk plan is carried through untouched.

When rescheduling does run, bindings that still match an existing plan entry by source, destination and
flags stay on their current device (affinity) and only the difference is charged. Anything with no match is
scheduled afresh. If a monopoly binding has any affinity, the whole monopoly group stays on its device.

The engine params report `volume_changed` whenever the resulting bind list differs from the old one, which
is core's signal that the workload has to be recreated rather than updated in place.

`calculate-remap` is a no-op: volumes are not moved between running workloads.

## Diffs and repair

`get-node-resource-info` recomputes node usage from the workload resources core passes in and reports every
disagreement: storage totals, per-volume sizes, and per-disk IOPS/BPS. `fix-node-resource` writes the
recomputed usage back when anything differs.

## Metrics

| Metric | Type | Labels | Value |
|---|---|---|---|
| `storage_used` | gauge | `podname`, `nodename` | bytes held by workloads |
| `storage_capacity` | gauge | `podname`, `nodename` | bytes the node offers |

The statsd keys are `core.node.<nodename>.storage.used` and `core.node.<nodename>.storage.capacity`, with
dots in the node name replaced by underscores.
