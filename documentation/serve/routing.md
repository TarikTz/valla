# Routing

## Single service

Proxy one local port to an auto-named HTTPS URL:

```bash
valla serve 5500
# → https://port5500.valla.test
```

If port 443 is taken the proxy falls back to 8443 then 9443. The printed URL always reflects the actual port.

## Named subdomains (`--map`)

Map specific subdomains to specific local ports:

```bash
valla serve --name film-portal --map "ui:3000,api:8080"
# → https://ui.film-portal.test   (→ :3000)
# → https://api.film-portal.test  (→ :8080)
```

Multiple entries are comma-separated: `"subdomain:port,subdomain2:port2"`.

## Port range (`--range`)

Auto-map a contiguous range of ports to `portN` subdomains:

```bash
valla serve --name dev --range 5500-5502
# → https://port5500.dev.test
# → https://port5501.dev.test
# → https://port5502.dev.test
```

## Combining flags

`--map` and `--range` can be used together:

```bash
valla serve --name myapp --map "web:3000" --range 8000-8001
# → https://web.myapp.test
# → https://port8000.myapp.test
# → https://port8001.myapp.test
```

## Declarative config (`valla.yaml`)

Place a `valla.yaml` file in your project directory and run `valla serve` with no arguments. CLI flags override file values.

```yaml
# valla.yaml
project: film-portal
domain: test
services:
  - name: web
    port: 5500
    subdomain: preview   # → https://preview.film-portal.test
  - name: api
    port: 8080
    subdomain: api       # → https://api.film-portal.test
  - name: sso
    port: 8443
    subdomain: auth      # → https://auth.film-portal.test
```

```bash
cd film-portal && valla serve
```

## LAN sharing (`--expose`)

Bind to `0.0.0.0` instead of `127.0.0.1` to make the proxy reachable by other devices on your local network:

```bash
valla serve --name myapp --map "ui:3000" --expose
```

A warning is printed with your LAN IP address. Use with caution — all configured routes become accessible to any device on the network.

## Streaming, WebSockets, and HMR

`valla serve` imposes no write timeout, so long-lived connections work without any configuration:

- **Vite / webpack HMR** — event streams stay open indefinitely
- **WebSockets** — HTTP Upgrade is forwarded transparently
- **SSE endpoints** — no deadline, no disconnections
- **Large file downloads** — not interrupted mid-transfer

## Unknown routes

Requests for a subdomain not in the routing table receive a `502` response with a plain-text list of configured routes.
