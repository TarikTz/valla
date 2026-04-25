# Dashboard (`--ui`)

Add `--ui` to any `valla serve` invocation to launch an interactive Bubbletea terminal dashboard instead of plain log output.

```bash
valla serve --name film-portal --map "ui:3000,api:8080" --ui
```

## What it shows

```
  Valla Proxy Active  |  film-portal  |  .test
  ────────────────────────────────────────────────────────────────────────
  #    STATUS      SUBDOMAIN     TARGET              HTTPS URL
  ────────────────────────────────────────────────────────────────────────
  1    ● ONLINE    ui            localhost:3000       https://ui.film-portal.test:8443
  2    ● ONLINE    api           localhost:8080       https://api.film-portal.test:8443
  ────────────────────────────────────────────────────────────────────────
  [Recent requests]
  GET    ui            /css/style.css           200  12ms
  POST   api           /v1/search               201  45ms
  ────────────────────────────────────────────────────────────────────────
  Press 1-9 to open in browser · q to quit
```

## Features

**Live health checks** — each service is polled every 4 seconds. Status shows `● ONLINE` or `○ DOWN` after the first check, `○ WAIT` before.

**Rolling request log** — the last 5 requests are shown with method, subdomain, path, status code, and latency.

**Keyboard shortcuts:**

| Key | Action |
|---|---|
| `1`–`9` | Open the corresponding service URL in your default browser |
| `q` / `Q` / `Ctrl-C` | Stop the proxy and exit |
