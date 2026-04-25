# Workflows

## Local workflow

Run the CLI and start each service directly on your machine.

```bash
npx valla-cli
cd my-app
```

Start the frontend:

```bash
cd frontend && npm install && npm run dev
```

Start the backend (command depends on your selected stack):

```bash
# Go
go run ./cmd/server

# Node.js
npm install && npm run dev

# Python
pip install -r requirements.txt && uvicorn main:app --reload
```

---

## Fully Dockerized workflow

Requires [VS Code](https://code.visualstudio.com/) with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers).

```bash
npx valla-cli          # choose "Fully Dockerized" in the output structure step
cd my-app
code .                 # VS Code prompts "Reopen in Container" — click it
```

Inside the container, install dependencies and start your dev server:

```bash
# Node / Bun
npm install && npm run dev

# Python
pip install -r requirements.txt && uvicorn main:app --reload --host 0.0.0.0
```

All packages are installed inside the container. Your host machine only ever sees the source files.

---

## Docker Compose workflow

For any output mode that includes a `docker-compose.yml`:

```bash
npx valla-cli
cd my-app
docker-compose up -d
```

---

## WordPress workflow

```bash
npx valla-cli          # choose "WordPress" in the output structure step
cd my-wordpress-project
docker-compose up -d
```

Open `http://localhost:<wordpress-port>` and complete the browser-based WordPress setup.

---

## Pairing scaffold with valla serve

After scaffolding, use `valla serve` to put your local services behind trusted HTTPS — eliminating CORS mismatches and mixed-content warnings without any additional configuration.

```bash
# Scaffold the project
npx valla-cli

# Set up HTTPS once per machine
npx valla-cli trust

# Start your dev servers, then proxy them
valla serve --name my-app --map "ui:3000,api:8080"
# → https://ui.my-app.test
# → https://api.my-app.test
```

See the [Serve docs](/serve/) for the full reference.
