# Frontend — Current State (P4)

Snapshot of the `client/` app after the P4 auth/RBAC/dark-mode/review/reports
work. Audience: anyone extending the frontend or integrating with it.

## Stack

| Concern | Choice |
|---|---|
| Framework | React 19 |
| Build / dev | Vite 6 |
| Language | TypeScript 5.8 (strict) |
| Styling | TailwindCSS v4 (Vite plugin, no `tailwind.config.js`) |
| Routing | react-router-dom v7 |
| Charts | recharts 3 |
| Tests | Vitest 3 + Testing Library + fast-check |
| Package manager | yarn 1.22 |

## Directory map

```
client/src/
├── App.tsx                 # Router, providers, role-aware nav
├── contexts/AuthContext.tsx# JWT session state (role, hasRole, login, logout)
├── theme/                  # ThemeContext, ThemeToggle, shared style tokens
├── components/
│   ├── ProtectedRoute.tsx  # auth gate + optional RBAC (roles=[...])
│   ├── AccessDenied/       # friendly 403 surface
│   ├── navbar/ button/     # chrome (dark-aware)
│   ├── ConfidenceBadge/ StatusBadge/
│   └── photosCards/ devicesCards/
├── hooks/
│   ├── useReviewQueue.ts   # list + approve/correct/reject (+ polling)
│   ├── useReviewSocket.ts  # forward-compatible /ws/review client
│   └── useReports.ts       # list, run, CSV export, result normaliser
├── pages/
│   ├── homePage/ loginPage/ register/
│   ├── photosPage/ devicesPage/ statisticsPage/   (skeleton, extended)
│   ├── reviewQueuePage/                            (new)
│   ├── reportsPage/ (+ ReportDetail.tsx)           (new)
│   └── adminPage/                                  (new)
├── types/                  # auth, review, report, user, photo
└── utils/                  # api (client), fields, format, stubs
```

## Auth flow

1. `loginPage` POSTs `/api/v1/auth/login` → `{ access_token, refresh_token,
   email, role }`. `AuthContext.login` persists them via `tokenStore`
   (localStorage).
2. `utils/api.ts` `apiFetch` attaches `Authorization: Bearer <access>` to every
   request. On `401` it does a single silent `/api/v1/auth/refresh` + retry; a
   second `401` clears the session and redirects to `/login`.
3. `logout` calls `/api/v1/auth/logout` (best effort) then clears storage.

Errors from the backend are **plain text** (`net/http` `http.Error`), so the
client reads `response.text()`, not JSON, for messages.

## RBAC

`AuthContext.role` drives both navigation visibility and `ProtectedRoute`
guards. Server-side enforcement still applies (`403` → `AccessDenied` / inline
message). Current client gating:

| Route | Roles |
|---|---|
| `/photos`, `/devices` | admin, doctor |
| `/statistics` | admin, doctor, researcher |
| `/reports`, `/reports/:name` | any authenticated (per-report check server-side) |
| `/review-queue` | admin, doctor |
| `/admin/users` | admin |

## Theming

Tailwind v4 class-based dark mode. `index.css` declares
`@custom-variant dark (&:where(.dark, .dark *))`; `ThemeProvider` toggles the
`.dark` class on `<html>` (persisted to `localStorage`, OS-preference fallback).
Use the tokens in `theme/styles.ts` (`card`, `input`, `label`, …) for new
surfaces so dark mode stays consistent.

## Data access

Plain hooks over the typed `apiJson`/`apiSend`/`postJson` helpers. No global
server-state cache yet (TanStack Query is a planned, optional upgrade).

## Testing

- `yarn test` — Vitest run (jsdom).
- `yarn test:pbt` — property-based suite only (fast-check; Lab 10 Ex 1).
- `yarn test:coverage` — v8 coverage.

## Running

```bash
cd client
yarn install
yarn dev            # http://localhost:5173 (proxies to API at :8080)
# VITE_API_BASE_URL overrides the backend URL.
```
