FROM oven/bun:1
WORKDIR /app
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile
COPY . .
EXPOSE 4200
CMD ["bunx", "ng", "serve", "--host", "0.0.0.0"]
