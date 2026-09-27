# Progress — ID Card Studio

## Tasks
1. [x] Check prerequisites, run `git init` (if needed), extract Appendix A into `reference/index.html` and Appendix B into `web/assets/brand/`, and add `.gitignore` (`dist/`, `data/`, `config.json`), `go.mod` and the skeleton layout.
2. [ ] Config loading and the portable data folder, with atomic JSON storage and tests.
3. [ ] Auth: PIN hashing, first-run setup, login and logout, sessions, lockout, role middleware, and tests.
4. [ ] Users API (list, add, reset PIN, enable/disable, last-admin guard), with tests.
5. [ ] Activity log: write, and query with filters and pagination.
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

## Blockers
(none yet)
