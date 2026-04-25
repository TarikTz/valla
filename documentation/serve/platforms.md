# Platforms & TLDs

## Choosing a domain suffix

The `--domain` flag (default: `test`) controls the TLD used for generated HTTPS URLs.

| Domain | Recommended | Notes |
|---|:---:|---|
| `test` | ✓ | Reserved for local use by RFC 2606. `valla trust` sets up wildcard DNS automatically on macOS and Linux. |
| `localhost` | ✓ | Always resolves to `127.0.0.1` — no DNS setup needed on any OS. |
| `lvh.me` | ✓ | Public wildcard DNS that always resolves to `127.0.0.1`. No local setup — works on any OS including Windows. Requires internet. |
| `dev` | ⚠ | Chrome and Edge enforce HSTS for `.dev`. The CLI prompts for confirmation before proceeding. |

## macOS

`valla trust` on macOS:

1. Generates the CA and installs it into the **System keychain** using `sudo security add-trusted-cert`.
2. Installs dnsmasq via Homebrew if not already present.
3. Adds `address=/.test/127.0.0.1` to the dnsmasq config.
4. Writes `/etc/resolver/test` (nameserver 127.0.0.1) using `sudo tee`.
5. Restarts the dnsmasq launchd service using `sudo launchctl`.

After `valla trust`, any `*.test` hostname resolves to `127.0.0.1` without any further configuration.

## Linux

`valla trust` tries three strategies in order, stopping at the first that succeeds:

1. **NetworkManager dnsmasq plugin** — writes `/etc/NetworkManager/dnsmasq.d/valla-test.conf` and reloads NetworkManager. Common on Ubuntu and Fedora desktop.
2. **systemd-resolved + dnsmasq:5353** — configures dnsmasq to listen on port 5353 and adds a systemd-resolved split-DNS drop-in that routes `.test` queries to it.
3. **Plain dnsmasq** — appends the rule to `/etc/dnsmasq.conf` and restarts the service.

If all three fail, the manual steps are printed and `valla trust` exits successfully. DNS failure is non-fatal.

## Windows

`valla trust` installs the CA into the Windows Root certificate store using `certmgr`. DNS is not configured automatically (no native wildcard resolver exists).

**Recommended:** use `--domain lvh.me` — no local DNS setup required.

```bash
valla serve 5500 --domain lvh.me
# → https://port5500.valla.lvh.me:8443
```

**Alternative:** install [Acrylic DNS Proxy](https://mayakron.altervista.org/wikibase/show.php?id=AcrylicHome), add `127.0.0.1 *.test` to `AcrylicHosts.txt`, and point your network adapter's DNS to `127.0.0.1`.
