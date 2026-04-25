Product Requirements Document: Valla-cli Secure Serve Module
============================================================

1\. Overview & Objective
------------------------

The **Secure Serve** module is a high-performance local reverse proxy built into `valla-cli`. It aims to eliminate the friction of local HTTPS and CORS configurations for full-stack environments. By providing zero-config SSL, custom domain namespacing, and multi-port subdomain routing, it allows developers to mirror production environments entirely locally.

2\. Core Features
-----------------

### 2.1 Zero-Config Local SSL

-   **Internal CA Generation:** On initialization, the CLI generates a persistent Root Certificate Authority (CA).

-   **System Trust:** A dedicated command installs the CA into the OS trust store, enabling the "Green Padlock" across browsers.

-   **Ephemeral Certificates:** Generates and caches wildcard certificates (e.g., `*.valla.test`) on the fly during the proxy session.

### 2.2 Multi-Service Orchestration & Routing

-   **Subdomain Routing:** Routes traffic to specific local ports based on the incoming request's subdomain (e.g., `api.myapp.test` $\rightarrow$ `localhost:8080`).

-   **Port Range Auto-Discovery:** Automatically maps a sequence of ports to sequentially named subdomains (e.g., `5500-5502` $\rightarrow$ `port5500`, `port5501`).

-   **Conflict Resolution:** If the default `443` port is bound by another process, the CLI automatically falls back to `8443` or `9443` and outputs the exact port in the terminal links.

### 2.3 Dynamic Namespacing & Configuration

-   **CLI Flags:** Rapidly spin up environments using inline arguments.

-   **Declarative Configuration (`valla.yaml`):** Persist multi-service architectures within the project repository for team sharing.

-   **DNS Loopback Support:** Native support for `.test`, `.localhost`, and public loopbacks like `.lvh.me` (which resolve to `127.0.0.1` without `/etc/hosts` modifications).

### 2.4 Optional Terminal Dashboard (`--ui` flag)

An optional, interactive CLI table view to monitor the health and traffic of multiple proxied services simultaneously.

* * * * *

3\. Command Line Interface (CLI) Specifications
-----------------------------------------------

### 3.1 Primary Commands & Flags

| **Command / Flag** | **Description** | **Example** |
| --- | --- | --- |
| `valla trust` | Installs the generated Root CA into the system trust store (requires `sudo`/admin once). | `npx valla-cli trust` |
| `serve [port]` | Starts the proxy for a single port on the default namespace. | `valla serve 5500` |
| `--name [val]` | Sets the base domain namespace to isolate projects. | `valla serve --name my-app` |
| `--map [val]` | Explicitly maps subdomains to local ports. | `valla serve --map "ui:3000,api:8080"` |
| `--range [val]` | Maps a continuous range of ports. | `valla serve --range 5500-5505` |
| `--ui` | **[Feature Toggle]** Replaces standard logs with the interactive Terminal Dashboard. | `valla serve --ui` |

* * * * *

4\. Configuration Schema (`valla.yaml`)
---------------------------------------

When executed, `valla serve` will automatically detect a `valla.yaml` in the current working directory. This overrides CLI flags unless explicitly passed.

YAML

```
# valla.yaml
project: film-portal
domain: test # Generates *.film-portal.test
services:
  - name: web
    port: 5500
    subdomain: preview  # https://preview.film-portal.test
  - name: api
    port: 8080
    subdomain: api      # https://api.film-portal.test
  - name: sso
    port: 8443
    subdomain: auth     # https://auth.film-portal.test

```

* * * * *

5\. Terminal Dashboard View (Triggered via `--ui`)
--------------------------------------------------

When the `--ui` flag is provided, standard scrolling logs are suppressed. The CLI utilizes ANSI escape sequences to render a persistent, auto-refreshing table.

### 5.1 UI Requirements

-   **Health Checks:** Pings targets every 3-5 seconds to update the Status column.

-   **Interactivity:** Pressing the corresponding number key (e.g., `1`, `2`) opens that URL in the default OS browser.

-   **Log Tailing:** A dedicated 5-line pane at the bottom displays only the most recent HTTP requests across all mapped ports.

### 5.2 Terminal Mockup

Plaintext

```
Valla Proxy Active | Project: film-portal | Base: .test
─────────────────────────────────────────────────────────────────────────────
[#]  STATUS    SUBDOMAIN    TARGET            HTTPS URL
─────────────────────────────────────────────────────────────────────────────
 1   ● ONLINE  preview      localhost:5500    https://preview.film-portal.test
 2   ● ONLINE  api          localhost:8080    https://api.film-portal.test
 3   ○ DOWN    auth         localhost:8443    https://auth.film-portal.test
─────────────────────────────────────────────────────────────────────────────
[Logs] 14:22:01 GET preview  /css/style.css -> 200 OK (12ms)
[Logs] 14:22:05 POST api     /v1/search     -> 201 Created (45ms)
─────────────────────────────────────────────────────────────────────────────
Press 1-3 to open in browser | 'q' to quit | 'r' to reload yaml

```

* * * * *

6\. Technical Architecture (Go Implementation)
----------------------------------------------

-   **Proxy Engine:** Utilize Go's `net/http/httputil.NewSingleHostReverseProxy` for lightweight, concurrent traffic forwarding.

-   **Dynamic TLS:** Implement a custom `tls.Config` with a `GetCertificate` hook. This reads the `ClientHelloInfo.ServerName` (SNI) and generates/serves the correct certificate in memory.

-   **Routing Map:** Maintain a `map[string]*httputil.ReverseProxy` where keys are the full hostnames (e.g., `api.film-portal.test`) to route incoming requests to the correct local target.

-   **Certificate Management:** Use `crypto/x509` and `crypto/rsa` or `crypto/ecdsa` to generate standard, compliant certificates with proper Subject Alternative Names (SANs).

7\. Security & Limitations
--------------------------

-   **HSTS Prevention:** The CLI should warn users if they attempt to map a `.dev` domain, as Chrome enforces strict HSTS for this TLD, which can break local development if certs are ever misconfigured. Default recommendations should strictly be `.test` or `.localhost`.

-   **Network Binding:** The proxy server must bind strictly to `127.0.0.1` (not `0.0.0.0`) to ensure local development environments are not accidentally exposed to the local area network (LAN) unless explicitly configured by the user.

-   **Privilege Constraints:** The proxy will attempt to bind to `443`. If it fails due to OS privilege restrictions (common on Linux/macOS without sudo), it must gracefully degrade to a high port (e.g., `8443`) rather than crashing.