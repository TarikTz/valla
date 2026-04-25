---
layout: home

hero:
  name: "Valla"
  text: "Scaffold your full stack in seconds."
  tagline: Pick your frameworks, hit Enter, get a production-ready project with environment config, Docker Compose, and local HTTPS — all wired up.
  image:
    src: /demo.gif
    alt: Valla CLI demo
  actions:
    - theme: brand
      text: Get started
      link: /scaffold/
    - theme: alt
      text: View on GitHub
      link: https://github.com/tariktz/valla

features:
  - icon: ⚡
    title: Full-stack scaffold
    details: Interactive TUI walks you through frontend, backend, database, and ORM selection. Generates a complete project with .env and Docker Compose wired up.
    link: /scaffold/
    linkText: Scaffold docs

  - icon: 🔒
    title: Zero-config local HTTPS
    details: valla serve turns any local port into a trusted HTTPS URL with a real certificate. No more CORS mismatches or mixed-content warnings.
    link: /serve/
    linkText: Serve docs

  - icon: 🐳
    title: Fully Dockerized dev environment
    details: Keep node_modules and Python venvs off your host machine entirely. Everything runs inside a Docker dev container — your source files are the only thing on disk.
    link: /scaffold/output-modes#fully-dockerized
    linkText: Learn more
---
