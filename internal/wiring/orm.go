package wiring

import (
	"bytes"
	"text/template"
)

const prismaSchemaTmpl = `generator client {
  provider = "prisma-client"
  output   = "../src/generated/prisma"
}

datasource db {
  provider = "{{.PrismaProvider}}"
}
`

const prismaConfigContent = `import 'dotenv/config';
import { defineConfig } from 'prisma/config';

export default defineConfig({
  schema: 'prisma/schema.prisma',
  migrations: {
    path: 'prisma/migrations',
  },
  datasource: {
    url: process.env.DATABASE_URL!,
  },
});
`

const drizzleConfigTmpl = `import 'dotenv/config';
import { defineConfig } from 'drizzle-kit';

export default defineConfig({
  out: './drizzle',
  schema: './src/db/schema.ts',
  dialect: '{{.DrizzleDialect}}',
  dbCredentials: {
    url: process.env.DATABASE_URL!,
  },
});
`

const drizzleIndexURLTmpl = `import 'dotenv/config';
import { drizzle } from 'drizzle-orm/{{.DrizzleImport}}';

const db = drizzle(process.env.DATABASE_URL!);

export { db };
`

const drizzleIndexSQLite = `import 'dotenv/config';
import { drizzle } from 'drizzle-orm/better-sqlite3';
import Database from 'better-sqlite3';

const db = drizzle(new Database(process.env.DATABASE_URL!, { uri: true }));

export { db };
`

type prismaData struct {
	PrismaProvider string
}

type drizzleData struct {
	DrizzleDialect string
	DrizzleImport  string
}

// GeneratePrismaConfig returns the content of prisma.config.ts for Prisma v7+.
// The datasource URL is always read from the DATABASE_URL environment variable.
func GeneratePrismaConfig() string {
	return prismaConfigContent
}

// GeneratePrismaSchema returns the content of prisma/schema.prisma for the given provider.
// provider is one of "postgresql", "mysql", or "sqlite".
func GeneratePrismaSchema(provider string) (string, error) {
	tmpl, err := template.New("prisma").Parse(prismaSchemaTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, prismaData{PrismaProvider: provider}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GenerateDrizzleConfig returns the content of drizzle.config.ts and src/db/index.ts.
// For dialect == "sqlite", importPath is ignored — the sqlite index.ts template is hardcoded.
// Callers should pass importPath = "" for sqlite.
func GenerateDrizzleConfig(dialect, importPath string) (string, string, error) {
	cfgTmpl, err := template.New("drizzle-cfg").Parse(drizzleConfigTmpl)
	if err != nil {
		return "", "", err
	}
	var cfgBuf bytes.Buffer
	if err := cfgTmpl.Execute(&cfgBuf, drizzleData{DrizzleDialect: dialect}); err != nil {
		return "", "", err
	}

	var idxContent string
	if dialect == "sqlite" {
		idxContent = drizzleIndexSQLite
	} else {
		idxTmpl, err := template.New("drizzle-idx").Parse(drizzleIndexURLTmpl)
		if err != nil {
			return "", "", err
		}
		var idxBuf bytes.Buffer
		if err := idxTmpl.Execute(&idxBuf, drizzleData{DrizzleImport: importPath}); err != nil {
			return "", "", err
		}
		idxContent = idxBuf.String()
	}

	return cfgBuf.String(), idxContent, nil
}
