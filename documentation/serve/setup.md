# Setup — valla trust

`valla trust` is a one-time, per-machine command. It generates a local root CA and installs it in your OS trust store so browsers show a green padlock for every certificate valla signs.

::: code-group

```bash [npx (no install)]
npx valla-cli trust
```

```bash [npm]
npm install -g valla-cli
valla trust
```

```bash [Homebrew]
# Coming soon
brew install valla-cli
valla trust
```

```bash [Binary]
./valla trust
```

:::

> **No `sudo` needed upfront.** `valla trust` invokes `sudo` internally for the specific steps that require it (trust store install, `/etc/resolver/` write). Running the whole command as root is not recommended.

## What it does

### 1. Generates a local CA

Creates `~/.valla/ca.pem` and `~/.valla/ca-key.pem` (ECDSA P-256, 10-year validity). If those files already exist and are valid, this step is skipped.

### 2. Installs the CA in your trust store

| Platform | Trust store | Command used |
|---|---|---|
| macOS | System keychain | `sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain` |
| Linux | NSS (Chrome/Firefox) + system | `certutil` + `update-ca-certificates` |
| Windows | Windows Root store | `certmgr /add ... /s Root` |

### 3. Configures wildcard DNS for `*.test → 127.0.0.1`

| Platform | Strategy |
|---|---|
| macOS | Installs dnsmasq via Homebrew, adds `address=/.test/127.0.0.1`, writes `/etc/resolver/test` |
| Linux | Tries NetworkManager dnsmasq plugin → systemd-resolved + dnsmasq:5353 → plain dnsmasq |
| Windows | Prints manual instructions — see [Windows DNS](#windows-dns) |

DNS setup is non-fatal. If it fails, `valla trust` prints a warning and the manual steps, then exits successfully. You can also skip DNS entirely by using `--domain lvh.me` with `valla serve`.

## Re-running trust

Running `valla trust` again is safe — it skips CA generation if the files already exist and re-applies the trust store and DNS configuration.

## Windows DNS

`valla trust` does not configure DNS on Windows automatically. Two options:

**Option A — No setup required (recommended):** use `--domain lvh.me` with `valla serve`. `lvh.me` is a public wildcard DNS entry that always resolves to `127.0.0.1`. No installation needed.

```bash
valla serve 5500 --domain lvh.me
# → https://port5500.valla.lvh.me:8443
```

**Option B — Acrylic DNS Proxy:** install [Acrylic](https://mayakron.altervista.org/wikibase/show.php?id=AcrylicHome), add `127.0.0.1 *.test` to `AcrylicHosts.txt`, and point your network adapter's DNS server to `127.0.0.1`.

## Port 443

Binding port 443 requires root on Linux and macOS. If `valla serve` is run without elevated privileges, it automatically falls back to **8443** then **9443**. The printed URL always reflects the actual port in use.
