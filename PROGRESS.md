# Progress — ID Card Studio

## Tasks
1. [x] Check prerequisites, run `git init` (if needed), extract Appendix A into `reference/index.html` and Appendix B into `web/assets/brand/`, and add `.gitignore` (`dist/`, `data/`, `config.json`), `go.mod` and the skeleton layout.
2. [x] Config loading and the portable data folder, with atomic JSON storage and tests.
3. [x] Auth: PIN hashing, first-run setup, login and logout, sessions, lockout, role middleware, and tests.
4. [x] Users API (list, add, reset PIN, enable/disable, last-admin guard), with tests.
5. [x] Activity log: write, and query with filters and pagination.
6. [ ] Presence tracking with reverse-DNS caching, and a server-info endpoint.
7. [ ] Clients API with SVG validation and sanitising, and the Helios seed. Tests.
8. [ ] Employees API (list, upsert, bulk import, map client) and the photo upload endpoint.
9. [ ] Export endpoint that saves PDFs into month folders and records exports. Tests.
10. [ ] `scripts/vendor.*`, fetching and committing the pinned libraries and fonts.
11. [ ] Frontend shell: embed, first-run setup, login, top bar, routing, API client and presence heartbeat.
12. [ ] Users page and Activity page.
13. [ ] Card renderer module, matching the reference exactly, with the live preview and navigation.
14. [ ] Wizard Step 1: clients dropdown, add/edit, SVG upload or paste, the Claude prompt, and live preview.
15. [ ] `scripts/make_template.py` and the committed Excel template.
16. [ ] Wizard Step 2: Excel/CSV import, the Needs attention mapping, the add-employee form, search and filter.
17. [ ] Wizard Step 3: photo queue, upload and resize, crop sliders and drag, save and next.
18. [ ] Wizard Step 4: filters, grouped selection with shift-range, and PDF/ZIP export with progress.
19. [ ] Under the hood panel.
20. [ ] `scripts/build.*`: cross-compile and test all four binaries.
21. [ ] End-to-end smoke test, README (setup, build, running on a LAN, backup by copying `data/`, the Windows OneDrive Desktop note, troubleshooting), cleanup pass, and the Handover section.

## Decisions
- Installed missing prerequisites without prompting: Go 1.27 via `winget install GoLang.Go`, `openpyxl` via `python -m pip install --user openpyxl`. Reason: rule 7 allows unattended install of missing tools.
- Go module name is `idcardstudio` (root `main.go`, no `cmd/` subdir) with `web` as a small embed-only package (`web.Assets`). Reason: simplest layout that supports `go:embed` of `web/assets` and later `web/static`/`web/vendor` without an extra indirection layer.
- Removed the two loose `wcf-logo-*.svg` files that were sitting at the repo root (they carried embedded C2PA metadata blobs, not byte-identical to Appendix B) and wrote the exact Appendix B markup to `web/assets/brand/` instead. Reason: spec requires the brand files saved "exactly as given"; the root copies weren't.
- `go.mod` declares `go 1.24` (the spec's minimum) even though the installed toolchain is 1.27. Reason: keep the minimum-version contract explicit; the installed newer toolchain still satisfies it.
- `internal/storage.Store[T]` is a generic mutex-protected JSON file wrapper with `Load`/`Save`/`Update`; `Update` holds the lock across the whole read-modify-write cycle so two concurrent API handlers can never race. Reason: the spec requires every write atomic *and* mutex-protected because several users work at once — a bare `Save` alone doesn't protect a read-then-write sequence done by the caller.
- Printing LAN URLs and auto-opening the browser is deferred to the frontend-shell task (11), not wired into today's minimal `main.go`. Reason: there's no UI to open yet; `main.go` for now just binds `0.0.0.0:<configured port>` and serves embedded assets.
- `crypto/pbkdf2` is a real Go 1.24+ stdlib package (confirmed with `go doc crypto/pbkdf2` against the installed 1.27 toolchain) — used directly, no third-party crypto module needed.
- Sessions live in memory only (`auth.SessionManager`), not persisted to `data/`. Reason: simplest option that fits the spec's 12-hour cookie requirement; a restart just signs everyone out, which is acceptable for a LAN tool restarted rarely.
- Lockout resets `failed_attempts` to 0 the moment the 5th failure trips `locked_until`, rather than continuing to count during the lock. Reason: simplest state machine that still satisfies "after 5 failed attempts, lock for 5 minutes" — attempts made *during* the lock are already rejected outright by the `locked_until` check before the PIN is even compared, so they can't further increment anything.
- Username matching is case-insensitive (`strings.EqualFold`) across lookup, create and update. Reason: spec makes client codes explicitly case-insensitive and is silent on usernames; treating them the same way avoids "ada" vs "Ada" duplicate-account confusion, a plausible real-world typo.
- `/api/setup` takes `{name, username, pin}` with no `confirm_pin` field — confirmation re-entry is a client-side-only UI check. Reason: the spec only requires confirm-PIN on the Users-page add-user form (task 4); keeping the setup endpoint's shape identical to `auth.Manager.Setup` avoids a mismatched, speculative field.
- `internal/server.Server.Handler()` returns routes un-prefixed (`GET /setup`, `POST /login`, …); `main.go` mounts it at `/api/` via `http.StripPrefix`. Reason: keeps the server package's route table independent of where it's mounted, and matches the `web/assets/` mount done the same way.
- The spec's "the last active admin can't be disabled or demoted" guard is only wired to the Disable action. Reason: the Users page spec lists exactly two actions per row — Reset PIN and Disable/Enable — with no role-change action anywhere in the UI; adding an unused "demote" endpoint now would be speculative code with no caller.
- `ResetPIN` also clears `failed_attempts` and `locked_until`. Reason: an admin resetting a PIN is already intervening on the account; leaving it locked after that would be a confusing dead end with no way to unlock it otherwise (there's no separate "unlock" action in the spec).
- The Users-list response adds a computed `status` field (`"active" | "disabled" | "locked"`) not present in the stored `User` struct. Reason: the spec's Users table has a "status" column but doesn't define its values; deriving it from `disabled`/`locked_until` is the simplest option and keeps the raw lockout timestamp out of the API response.
- `resolveHost` (in `internal/server/netinfo.go`) does a plain, uncached `net.LookupAddr` with a 500ms timeout for every activity-log write. Reason: task 5 (activity log) comes before task 6 (presence's cached reverse-DNS); activity writes are infrequent (sign-ins, admin actions) so the extra latency is acceptable, and task 6 can later share or replace this with the cached lookup used for presence's every-15-seconds heartbeat, which really does need caching.
- `GET /api/activity` is available to any signed-in user (`RequireAuth`), not Admin-only. Reason: the top bar spec marks only the Users tab "(Admin only)"; Activity is listed alongside Wizard with no such qualifier.
- The Activity page's `from`/`to` filters take plain `YYYY-MM-DD` dates (parsed as UTC, `to` extended to end-of-day) rather than full timestamps. Reason: spec describes the filter as "date range," and the fixed English month-list precedent elsewhere in the spec favors plain calendar dates over timezone-sensitive timestamps.

## Blockers
(none yet)
