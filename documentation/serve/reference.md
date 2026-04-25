# Flag reference

## valla trust

No flags. Run once per machine.

```bash
npx valla-cli trust
```

## valla serve

```bash
valla serve [<port>] [flags]
```

| Flag | Default | Description |
|---|---|---|
| `<port>` | — | Single-port mode: proxy `localhost:<port>` to an auto-named subdomain |
| `--name <ns>` | `valla` | Subdomain namespace. URLs are `*.{name}.{domain}` |
| `--domain <tld>` | `test` | Domain suffix. Use `test`, `localhost`, or `lvh.me`. `.dev` triggers a confirmation prompt |
| `--map <pairs>` | — | Comma-separated `subdomain:port` pairs, e.g. `"ui:3000,api:8080"` |
| `--range <range>` | — | Port range for auto-mapping, e.g. `5500-5502` → `port5500`, `port5501`, `port5502` |
| `--ui` | off | Launch the interactive Bubbletea dashboard instead of plain log output |
| `--expose` | off | Bind to `0.0.0.0` instead of `127.0.0.1` for LAN sharing |

## valla.yaml

When `valla serve` is run in a directory containing `valla.yaml`, the routing table is loaded from the file. Explicit CLI flags always override file values.

```yaml
project: my-app       # maps to --name
domain: test          # maps to --domain
services:
  - name: web         # human label (display only)
    port: 3000
    subdomain: ui     # → https://ui.my-app.test
  - name: api
    port: 8080
    subdomain: api    # → https://api.my-app.test
```

## Port fallback

`valla serve` tries ports in this order: **443 → 8443 → 9443 → OS-assigned**. The first available port is used and the printed URLs always reflect the actual port. Port 443 requires elevated privileges on Linux and macOS.

## URL format

| Proxy port | Example URL |
|---|---|
| 443 | `https://ui.myapp.test` |
| 8443 | `https://ui.myapp.test:8443` |
| 9443 | `https://ui.myapp.test:9443` |
| other | `https://ui.myapp.test:<port>` |
