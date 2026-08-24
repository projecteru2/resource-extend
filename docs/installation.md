# Installation

## Build from source

Go 1.27 or newer:

```bash
git clone https://github.com/projecteru2/resource-extend.git
cd resource-extend
make build
```

`make build` writes two static binaries into the repository root:

| Binary | Package | Schedules |
|---|---|---|
| `resource-gpu` | `./cmd/resource-gpu` | GPU cards, by product model |
| `resource-storage` | `./cmd/resource-storage` | volumes, disk IOPS/BPS, storage space |

Cross compiling is a matter of `GOOS`/`GOARCH`; the plugins are pure Go with `CGO_ENABLED=0`. Release
archives for linux and darwin on amd64 and arm64 are published by goreleaser on every tag.

## Release archives

```bash
curl -sL https://github.com/projecteru2/resource-extend/releases/latest/download/resource-extend_Linux_x86_64.tar.gz \
  | tar -xz -C /etc/eru/plugins resource-gpu resource-storage
```

## Wiring into core

Core loads every **executable** file in `resource_plugin.dir` as a binary plugin, so the binaries must be
installed mode 0755. The resource name a node reports is the binary's file name, so keep the names
`resource-gpu` and `resource-storage` unless you are deliberately renaming the resource.

```yaml
# /etc/eru/core.yaml
resource_plugin:
  dir: /etc/eru/plugins
  call_timeout: 30s
```

`resource_plugin.whitelist` is unrelated to loading: when it is set, core only asks the plugins it names for
node resource info. Leave it unset unless you are deliberately hiding a resource from `node get`.

```bash
install -m 0755 resource-gpu resource-storage /etc/eru/plugins/
install -m 0644 gpu.yaml.sample     /etc/eru/plugins/gpu.yaml
install -m 0644 storage.yaml.sample /etc/eru/plugins/storage.yaml
systemctl restart eru-core
```

Core runs each plugin with its working directory set to `resource_plugin.dir`, so the default relative
config paths (`gpu.yaml`, `storage.yaml`) resolve next to the binaries. Set
`ERU_RESOURCE_CONFIG_PATH` in core's environment to override both, or pass `--config` when running a
plugin by hand.

## Verifying

Every verb reads JSON from stdin and writes JSON to stdout, so a plugin can be exercised directly:

```bash
cd /etc/eru/plugins
./resource-gpu --version
echo '{"nodename": "node-1"}' | ./resource-gpu get-node-resource-info
echo '{"podname": "dev", "nodename": "node-1"}' | ./resource-storage get-metrics
```

A non-zero exit status means the call failed; the error text goes to stderr, which core folds into the
plugin output it logs.
