# What is valla serve?

`valla serve` is a local TLS reverse proxy built into the CLI. It gives every local service a real HTTPS URL with a trusted certificate — eliminating CORS mismatches and mixed-content browser warnings during full-stack development.

## Why it exists

Modern browsers enforce strict CORS and HTTPS policies that make local development with multiple services painful:

- A frontend on `http://localhost:3000` calling an API on `http://localhost:8080` triggers CORS preflight on every request.
- Service workers, web push, and payment APIs refuse to run over plain HTTP.
- Mixed-content warnings block HTTP sub-resources on HTTPS pages.

`valla serve` solves all of this by routing each service to its own `https://subdomain.namespace.test` URL, with a certificate signed by a locally-trusted CA.

## How it works

1. **`valla trust`** (one-time, per machine) — generates a root CA, installs it in your OS trust store, and configures wildcard DNS so `*.test → 127.0.0.1`.
2. **`valla serve`** — starts a TLS reverse proxy that routes incoming requests to your local services based on the subdomain.

```
Browser → https://ui.myapp.test:8443
            ↓
       valla proxy (TLS termination)
            ↓
       http://127.0.0.1:3000
```

Certificates are generated on demand using ECDSA P-256 and signed by the local CA. The CA lives in `~/.valla/` and is created once.

## Quickstart

**One-time setup:**

::: code-group

```bash [npx]
npx valla-cli trust
```

```bash [Installed]
valla trust
```

:::

**Proxy a single service:**

::: code-group

```bash [npx]
npx valla-cli serve 3000
# → https://port3000.valla.test
```

```bash [Installed]
valla serve 3000
# → https://port3000.valla.test
```

:::

**Proxy multiple named services:**

::: code-group

```bash [npx]
npx valla-cli serve --name myapp --map "ui:3000,api:8080"
# → https://ui.myapp.test
# → https://api.myapp.test
```

```bash [Installed]
valla serve --name myapp --map "ui:3000,api:8080"
# → https://ui.myapp.test
# → https://api.myapp.test
```

:::

See [Setup](/serve/setup) to configure trust and DNS, and [Routing](/serve/routing) for all proxy options.
