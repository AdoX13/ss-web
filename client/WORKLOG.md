# Frontend (P4) — Worklog

Role: **P4 — Frontend Engineer** (`MedSecOCR_TeamPlan_v3.md` §10 P4)
Codebase: `client/` (React 19 + Vite + TypeScript + TailwindCSS v4)

---

## 2026-05-24 — Auth migration, RBAC, dark mode, review queue, reports, admin

Worked against the live backend described in the P1 handoff (all `/api/v1/*`
routes verified directly in `server/routes/`).

### Done

**Foundation / auth (handoff §1, §2, §9)**
- Rewrote `utils/api.ts`: `apiFetch` now attaches the bearer token, performs a
  single silent `/api/v1/auth/refresh` + retry on `401`, and redirects to
  `/login` on hard failure. Added typed `apiJson` / `apiSend` / `postJson`
  helpers (backend returns **plain-text** errors, handled accordingly).
- Replaced the guest auto-login stub in `contexts/AuthContext.tsx` with real
  JWT auth: stores access + refresh tokens, email, and role; exposes `role`,
  `isAdmin`, and `hasRole(...)`.
- `components/ProtectedRoute.tsx`: real auth gate plus optional `roles` RBAC;
  renders `AccessDenied` when the role is insufficient.
- `loginPage` / `register` migrated from legacy `/login` `/register` to
  `/api/v1/auth/*`; client-side min-8 password check; plain-text error parsing.
- Added `types/` (auth, review, report, user, photo) and
  `utils/fields.ts` (FIELD_LABELS + enum options from handoff §6).

**Dark mode (Phase B)**
- Tailwind v4 class-based dark mode via `@custom-variant dark` in `index.css`.
- `theme/ThemeContext.tsx` (persisted + OS fallback) and `theme/ThemeToggle.tsx`
  in the navbar. Shared style tokens in `theme/styles.ts`.
- Dark pass across navbar, button, home, login, register, photos, devices,
  statistics, and the photo/device cards.

**Review queue (Phase C)**
- `pages/reviewQueuePage/` + `hooks/useReviewQueue.ts`: list with status/field
  filters, approve/correct/reject, enum `<select>` for `control_type` /
  `medical_opinion`, confidence + status badges, audit hint, keyboard shortcuts
  (j/k/a/r/c), auto-refresh polling, dev-only sample data.
- `hooks/useReviewSocket.ts`: forward-compatible `/ws/review` client (see
  ISSUES below); the queue stays correct via polling regardless.

**Reports (Phase D)**
- `pages/reportsPage/` (landing) + `ReportDetail.tsx`: date range, run, generic
  table, auto bar-chart for (label, number) results, **auth-aware CSV export**
  (downloads via `apiFetch` because the bearer token can't ride on `<a download>`).
- `hooks/useReports.ts` normalises both the lower-case list shape and the
  capitalised run shape (see ISSUES).

**Admin (Phase E)**
- `pages/adminPage/`: list users, create, change role, deactivate (admin only).

**RBAC navigation + routing**
- `App.tsx`: `ThemeProvider`, role-aware nav, and route guards
  (Photos/Devices = admin+doctor; Statistics = +researcher; Review Queue =
  admin+doctor; Admin = admin; Reports = any authed).

**Testing (Phase H, partial)**
- Vitest + Testing Library + fast-check. 26 tests across utils/hooks/types and
  one component. `yarn test:pbt` runs the property-based suite (Lab 10 Ex 1).

### Gates (all green)
- `yarn lint` — 0 errors, 0 warnings
- `yarn build` — passes (`tsc -b && vite build`)
- `yarn test` — 26 passing · `yarn test:pbt` — 14 passing

### Decisions / deviations from the plan
- **Package manager: yarn 1.22** (matches `yarn.lock`, README, and CI). A stale
  `package-lock.json` remains from the skeleton — recommend deleting it to avoid
  mixed-lockfile drift (left it untouched for now).
- **No TanStack Query / Zod yet.** Used lightweight custom hooks + manual
  guards to stay consistent with the skeleton and keep the diff reviewable.
  Easy to adopt later.
- **No `lucide-react`** — inline SVGs match the skeleton's style.
- Branched off `main` as `feature/frontend-p4` (no `develop` branch exists in
  the remote).

### Next
- Phase F — Camera capture page: **blocked** on `POST /api/v1/documents`
  (not implemented) and on P3's page. See ISSUES.
- Phase G — Accessibility pass (axe-core via Playwright, full keyboard nav,
  Lighthouse ≥ 90).
- Phase H — Playwright E2E across the 4 roles; raise coverage toward ≥ 75%
  (currently only the util/hook/type/PBT layer is covered).
- Generate the API client from `server/docs/openapi.yaml` once P1 commits it.

See `docs/development/frontend_p4_contract_issues.md` for backend asks.
