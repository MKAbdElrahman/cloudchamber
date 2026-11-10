# Bubble

Web app for CloudChamber.

Modular React + Vite + TypeScript + Tailwind v4 + **[Meta Astryx](https://github.com/meta/astryx)** (`@astryxdesign/core`, Neutral theme) + TanStack Query + `tus-js-client`.

Mirrors backend modularity by **feature/domain** modules:

- `src/features/auth` — login / JWT (auth service token or paste token)
- `src/features/upload` — resumable tus upload to tusd (owner-scoped, verified server-side)
- `src/features/pipeline` — create pipeline jobs from a verified `sourceRef` and view status
- `src/shared` — config/env, API client, session, layout chrome (`AppShell` / `TopNav`)
- `src/app/AstryxProviders.tsx` — theme + layer providers
- `references/astryx/` — upstream template mapping + `pnpm astryx` CLI usage

Astryx surfaces in use: resizable job **console** (`Table`, inspector panel), **toasts**, **Stepper** wizard, split **login**, **MetadataList** settings. Legacy shadcn `shared/ui` removed.

## MVP scope

Upload-only (no URL ingest) → create job → view job status. “My uploads” list and export/artifact download are not included (backend endpoints/processing work required).

## Configure

Copy `.env.example` to `.env` and adjust bases for your environment (kind dev defaults shown). The auth token endpoint is `GET {VITE_AUTH_API_BASE}/v1/auth/token/{VITE_AUTH_KID}` (Basic auth). You can also paste an existing bearer token.

## Run

```bash
pnpm install
pnpm dev
```

Build: `pnpm build`. Quality gate: `pnpm check` (lint, unit tests, build, `astryx doctor`).

Routes when signed in: `/overview`, `/pipeline`, `/settings/appearance` (and `account`, `developer`).
