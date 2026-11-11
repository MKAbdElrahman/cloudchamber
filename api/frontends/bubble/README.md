# Bubble

Web app for CloudChamber.

Modular React + Vite + TypeScript + Tailwind v4 + **[Meta Astryx](https://github.com/meta/astryx)** (`@astryxdesign/core`) + TanStack Query + `tus-js-client`.

Sign in with Google. A project holds its recordings and documents. Settings are Appearance and Account.

Feature modules:

- `src/features/auth` — Google sign-in
- `src/features/projects` — projects, and the files in a project
- `src/features/upload` — resumable tus upload to tusd
- `src/features/pipeline` — start a conversion from a file and follow it
- `src/features/outputs` — the document a job writes back into the project
- `src/features/settings` — Appearance and Account
- `src/shared` — config, API client, session, side-nav chrome
- `src/app/AstryxProviders.tsx` — theme providers

## Configure

Copy `.env.example` to `.env` and adjust the bases for your environment. Sign-in sends the browser to `{VITE_AUTH_API_BASE}/api/auth/google`.

## Run

```bash
pnpm install
pnpm dev
```

Build: `pnpm build`. Quality gate: `pnpm check` (lint, unit tests, build, `astryx doctor`).

Routes when signed in: `/` for projects, `/p/:projectId` for one project, `/settings/appearance`, and `/settings/account`.
