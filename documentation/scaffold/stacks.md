# Supported stacks

## Frontend

| Framework | Node | Bun | Server-side rendering |
|---|:---:|:---:|:---:|
| React | ✓ | ✓ | |
| Vue | ✓ | ✓ | |
| Angular | ✓ | ✓ | |
| Svelte (SvelteKit) | ✓ | ✓ | ✓ |
| Astro | ✓ | ✓ | ✓ |
| Next.js | ✓ | ✓ | ✓ |
| TanStack Start | ✓ | ✓ | ✓ |

> Server-side frameworks (SvelteKit, Astro, Next.js, TanStack Start) are eligible for ORM integration in frontend-only mode.

## Backend

| Language | Framework / template |
|---|---|
| Go | Gin |
| Go | Fiber |
| Go | Boilerplate |
| Node.js | Express |
| Node.js | NestJS |
| Node.js | Boilerplate |
| Python | FastAPI |
| Python | Flask |
| Python | Django |
| .NET | ASP.NET Core (Web API) |
| .NET | Minimal API |
| Java | Spring Boot (Maven) |
| Java | Spring Boot (Gradle) |
| Java | Quarkus (Maven) |
| Java | Quarkus (Gradle) |

## Database

Multiple databases can be selected simultaneously. SQLite and None are exclusive options.

| Database | Docker service | Combinable |
|---|:---:|:---:|
| PostgreSQL | ✓ | ✓ |
| MySQL | ✓ | ✓ |
| MariaDB | ✓ | ✓ |
| MongoDB | ✓ | ✓ |
| Redis | ✓ | ✓ |
| SQLite | | exclusive |

## ORM

Shown when at least one SQL database is selected and the project uses a Node.js runtime (Node backend or a server-side frontend framework).

| ORM | Generated files |
|---|---|
| Prisma | `prisma/schema.prisma`, `prisma.config.ts` |
| Drizzle | `drizzle.config.ts`, `src/db/index.ts` |
