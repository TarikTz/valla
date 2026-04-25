# Output modes

## Fully Dockerized

Valla's standout mode. Instead of installing project dependencies (`node_modules`, Python virtual envs) directly on your machine, everything runs inside a Docker dev container. Your source code is the only thing on your host disk — packages never leave the container.

**Why this matters for security:** when a compromised package runs a malicious `postinstall` script, it executes inside the container and cannot reach your SSH keys, home directory, or system files. The blast radius is contained.

**Generated structure:**

```text
my-app/
├── frontend/
├── backend/
├── .devcontainer/
│   └── devcontainer.json   ← open in VS Code to enter the container
├── docker-compose.dev.yml  ← named volumes shadow node_modules / .venv
├── docker-compose.yml
├── Makefile
└── .env
```

Open the project in VS Code and click **Reopen in Container** — the entire dev environment starts inside Docker. Ports are forwarded automatically and the relevant language extension is pre-installed inside the container.

> Requires Docker. The option is only shown when Docker is detected on your machine.

---

## Monorepo

Frontend and backend in a single directory, sharing one `.env` and `docker-compose.yml`.

```text
my-app/
├── frontend/
├── backend/
├── .env
└── docker-compose.yml
```

With Prisma selected:

```text
my-app/
├── frontend/
├── backend/
│   ├── prisma/
│   │   └── schema.prisma
│   └── prisma.config.ts
├── .env               ← includes DATABASE_URL
└── docker-compose.yml
```

---

## Separate folders

Frontend and backend generated as independent sibling directories.

```text
my-app-frontend/
my-app-backend/
.env
docker-compose.yml
```

---

## Frontend only

Generates only the frontend directory with your chosen framework and runtime. If a server-side framework (Next.js, SvelteKit, Astro, TanStack Start) is selected, ORM integration is offered.

---

## Backend only

Generates only the backend directory with your chosen language and framework.

---

## WordPress

Downloads the latest WordPress source, prepares Docker services, and creates a starter theme.

```text
my-wordpress-project/
├── .env
├── docker-compose.yml
└── wordpress/
    └── wp-content/
        └── themes/
            └── my-wordpress-project/
```

Start it up:

```bash
cd my-wordpress-project
docker-compose up -d
```

Then open `http://localhost:<wordpress-port>` in your browser and complete the WordPress setup wizard.
