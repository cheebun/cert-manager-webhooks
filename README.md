# cert-manager-webhooks

A monorepo of [cert-manager](https://cert-manager.io/) DNS01 webhook solvers for multiple DNS providers.

## Providers

| Provider | Directory |
|----------|-----------|
| [Alibaba Cloud DNS](https://www.alibabacloud.com/product/dns) | `webhooks/alidns/` |
| [NameSilo](https://www.namesilo.com/) | `webhooks/namesilo/` |
| [Spaceship](https://www.spaceship.com/) | `webhooks/spaceship/` |
| [Tencent Cloud DNSPod](https://cloud.tencent.com/product/cns) | `webhooks/tencent/` |

## Architecture

Each provider is an independent Go module with its own `go.mod`, Dockerfile, and Helm chart. A Go workspace (`go.work`) enables local cross-module development.

```
webhooks/<provider>/      # One module per provider
├── main.go               # Entrypoint
├── <provider>/solver.go  # DNS01 Present/CleanUp
├── Dockerfile
└── .goreleaser.yml
charts/<provider>-webhook/ # Helm chart
```

## Quick Start

```bash
go work sync
cd webhooks/alidns && go run .
```

## License

[Apache License 2.0](LICENSE)
