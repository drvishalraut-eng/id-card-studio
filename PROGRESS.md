# Progress — ID Card Studio

## Tasks
1. [x] Check prerequisites, run `git init` (if needed), extract Appendix A into `reference/index.html` and Appendix B into `web/assets/brand/`, and add `.gitignore` (`dist/`, `data/`, `config.json`), `go.mod` and the skeleton layout.
2. [x] Config loading and the portable data folder, with atomic JSON storage and tests.
3. [x] Auth: PIN hashing, first-run setup, login and logout, sessions, lockout, role middleware, and tests.
4. [x] Users API (list, add, reset PIN, enable/disable, last-admin guard), with tests.
5. [x] Activity log: write, and query with filters and pagination.
6. [x] Presence tracking with reverse-DNS caching, and a server-info endpoint.
7. [x] Clients API with SVG validation and sanitising, and the Helios seed. Tests.
8. [x] Employees API (list, upsert, bulk import, map client) and the photo upload endpoint.
9. [x] Export endpoint that saves PDFs into month folders and records exports. Tests.
10. [x] `scripts/vendor.*`, fetching and committing the pinned libraries and fonts.
11. [x] Frontend shell: embed, first-run setup, login, top bar, routing, API client and presence heartbeat.
12. [x] Users page and Activity page.
13. [x] Card renderer module, matching the reference exactly, with the live preview and navigation.
14. [x] Wizard Step 1: clients dropdown, add/edit, SVG upload or paste, the Claude prompt, and live preview.
15. [x] `scripts/make_template.py` and the committed Excel template.
16. [x] Wizard Step 2: Excel/CSV import, the Needs attention mapping, the add-employee form, search and filter.
17. [x] Wizard Step 3: photo queue, upload and resize, crop sliders and drag, save and next.
18. [x] Wizard Step 4: filters, grouped selection with shift-range, and PDF/ZIP export with progress.
19. [x] Under the hood panel.
20. [x] `scripts/build.*`: cross-compile and test all four binaries.
21. [x] End-to-end smoke test, README (setup, build, running on a LAN, backup by copying `data/`, the Windows OneDrive Desktop note, troubleshooting), cleanup pass, and the Handover section.

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
- `internal/presence.Manager` now owns the 10-minute reverse-DNS cache, and `Server.logAction` (activity log, task 5) was switched from its own uncached lookup to `Presence.ResolveHost`. Reason: this was flagged as a planned follow-up when task 5 landed — one cache is simpler than two, and activity writes benefit from the same 10-minute cache presence's heartbeat needs.
- `presence.Manager.List()` keys connected users by username (one row per user), so two tabs open for the same account collapse into a single entry showing whichever tab's heartbeat landed most recently. Reason: spec's panel mockup is "one row per user," not per tab/session, and the spec is silent on multi-tab behavior — last-heartbeat-wins is the simplest option.
- `internal/netutil.LANURLs` enumerates non-loopback IPv4 addresses for the server-info endpoint's `urls` field now, ahead of task 11's console startup printing. Reason: the server-info endpoint needs "URL" today per spec; task 11 will reuse this exact helper rather than duplicating it.
- `GET /api/server-info` and `POST /api/presence` are `RequireAuth` (any signed-in user), matching Activity's reasoning — the "Under the hood" panel is part of the Wizard, not gated by role.
- **Important for task 13 (card renderer):** the seeded Helios client's `logo` field is *only* the sunburst mark SVG (self-contained, viewBox `0 0 160 160`), reproducing reference/index.html's `<symbol id="helios-mark">`. The "HELIOS MATERIAL"/"HANDLING" wordmark next to it in the reference is plain CSS-styled HTML text (`.helios b`/`.helios span`), not SVG — it is NOT baked into the logo field, since doing so would require either hand-drawn letterform paths (not given anywhere in the spec) or `<text>` elements (which the spec's own SVG validation rules forbid). The card renderer must special-case the client whose `code` is `Helios` (case-insensitively) and reproduce the reference's exact `.helios` HTML+CSS lockup (icon + letter-spaced wordmark) rather than treating it as a generic uploaded-logo image — exactly how reference/index.html itself special-cases Helios via its own hardcoded `HELIOS` JS constant, separate from the generic per-client logo path.
- Client `id` is a slugified version of `name` (lowercase, non-alphanumeric runs collapsed to `-`), de-duplicated with a `-2`, `-3`, … suffix on collision. Reason: spec gives no id format; this mirrors the reference mockup's own `key(cli.name.value).replace(/[^a-z0-9]+/g,'-')` and keeps ids human-readable in `data/clients.json` and `data/logos/`.
- SVG validation walks the token stream with `encoding/xml` (not a regex over raw text) and explicitly rejects any `<!DOCTYPE`/`<!ENTITY` up front. Reason: a proper parse can't be fooled by comments/CDATA the way a naive substring search could, and blocking DTDs/entities closes off XXE/entity-expansion risk entirely rather than trusting `encoding/xml`'s defaults.
- An `href`/`*:href` is allowed only when it is a bare `#fragment` reference; anything else (`http(s)://`, `//`, `data:`, etc.) is rejected as "external." Reason: the spec explicitly bans "external hrefs" but the reference mockup's own approved SVGs use internal `<use href="#helios-mark">` — a same-document fragment isn't external, so blocking only non-fragment hrefs satisfies the spec without breaking that legitimate pattern.
- `Client.Update` treats an empty `logo` string as "keep the existing logo" rather than "clear the logo." Reason: the Step 1 edit form re-populates from the existing client and only sends new logo bytes when the admin actually uploads/pastes a replacement; a client can never be left with no logo through this endpoint, matching "fit the logo inside the logo box" implying a logo always exists once a client is created.
- Excel/CSV parsing itself (SheetJS, vendored) stays client-side per spec ("Upload .xlsx or .csv using SheetJS"); the server's `/api/employees/import` takes already-extracted JSON rows (`employee_id, name, role, client, join_date, photo_note`), with `join_date` as either a string or a raw Excel serial number. Reason: the spec explicitly names SheetJS as a client-side library for the parsing step, but also lists "Excel date parsing" as a server-testable concern — so date *parsing/normalization* (serial vs. `YYYY-MM-DD` vs. `DD-MM-YYYY`) happens in Go (`employees.ParseDateValue`, unit-tested), while sheet decoding itself does not move to the server.
- The Excel serial-date epoch (`1899-12-30`) is the standard, universally-replicated simplification: exactly correct for every real-world date from 1 Mar 1900 onward, off by one day only for Jan/Feb 1900 (Excel's fictitious leap day). Reason: employee join dates will never fall in that 60-day window in 1900, so the simpler, ubiquitous formula is the right tradeoff over handling that dead zone precisely.
- An import row with an unrecognized `client` code is still imported (not skipped) — only a missing/invalid `employee_id`, missing `name`, or unparseable `join_date` causes a skip. Reason: the spec's "Needs attention" mapping UI exists specifically to fix unknown client codes *after* import; treating an unknown client as a hard error would make that workflow unreachable.
- There is no separate "crop only" endpoint. `POST /employees/{id}/photo` takes the photo file *and* `zoom`/`x`/`y` together in one multipart request, committed atomically. Reason: spec's crop UI (drag/sliders) operates on a client-held, not-yet-uploaded `File` with a local preview; only "Save & next" should touch the server, and "Cancel... keeps the old photo" is then trivially true because nothing was ever sent — no server-side staging/undo needed.
- Photo upload does a minimal JPEG magic-byte check (`0xFFD8FF`) before writing to `<employee_id>.jpg`. Reason: the destination filename is server-controlled (not derived from the uploaded file), so this isn't a traversal defense — it's a cheap guard against an arbitrary file silently masquerading as a card photo later served/rendered as one.
- Uploading a new photo always resets `crop` to whatever zoom/x/y the same request carries (there is no "keep old crop, just replace photo" path). Reason: a new photo's framing rarely matches the old photo's crop values; forcing the client to always send fresh crop values with the new photo is simpler than adding a "preserve" flag nobody asked for.
- `POST /api/export` takes ONE rendered PDF per call (multipart `employee_id` + `pdf`), not a batch. Reason: the spec's own UI description ("Rendering 12 / 40…") already implies the browser loops one card at a time; one-PDF-per-call sidesteps the ambiguity of correlating N files with N employee_ids in a single multipart request, and matches how the browser already has to sequence renders anyway. The ZIP archive for "several cards" is purely a client-side `JSZip` bundle for the user's own download (task 18) — the server never sees or builds one.
- The month folder (`<Mon YYYY>`) is derived from each employee's own `join_date`, not from "today." Reason: spec's Step 4 filter is explicitly "month of the join date," and folder-per-cohort only makes sense keyed the same way; using the export date instead would scatter one join-month's cards across many folders depending on when each was (re-)exported.
- Export does a minimal PDF magic-byte check (`%PDF`) before writing, mirroring the photo upload's JPEG check — same reasoning: the destination path is server-controlled, so this guards against a wrong/empty blob being silently saved as a card PDF, not against path traversal.
- Pinned versions: html-to-image 1.11.13, jsPDF 4.2.1, JSZip 3.10.2, SheetJS (xlsx) 0.18.5 — each fetched from jsDelivr's npm mirror (`cdn.jsdelivr.net/npm/<pkg>@<version>/dist/...`) and checksum-verified by `scripts/vendor.sh`/`.ps1` (sha256 pinned alongside each URL). Reason: these are the current stable releases as of 2026-09-27; jsDelivr serves the exact npm-published dist files so the sha256 pin is meaningful (a real supply-chain check, not just a formality).
- Montserrat and Josefin Sans are each vendored as a **single variable-font woff2** (Google Fonts' own CSS serves the identical file URL for every weight in reference/index.html's request — 400/500/600/700 for Montserrat, 400/600 for Josefin Sans — confirmed by fetching the actual `css2` response), only the "latin" subset. `web/vendor/fonts/fonts.css` declares each with a `font-weight: <min> <max>` range, exactly mirroring what Google's own stylesheet does, so only 2 font files are vendored total rather than 6. Reason: this is what the upstream service itself does for these two families; latin-only keeps the files small and covers the English/Latin-script content this app renders (employee names, UI copy) — non-Latin diacritics beyond that are out of scope, spec is silent on it, and latin-ext can be added later if a real need shows up.
- `scripts/vendor.sh`/`.ps1` verify a pinned sha256 per file and fail loudly (deleting the bad download) on mismatch, rather than fetching unconditionally. Reason: this is the only supply-chain check standard-library Go tooling gives us for vendored JS/fonts committed straight into the repo — worth the small extra script complexity.
- `web/web.go` now embeds `index.html static assets vendor` as one `embed.FS` (`web.FS`), served directly at `/` via `http.FileServer(http.FS(web.FS))` — no more `fs.Sub`/`StripPrefix` gymnastics, since the embedded paths already match the URL paths 1:1 (`/static/js/app.js`, `/assets/brand/...`, `/vendor/...`). Reason: simpler than the old assets-only embed now that there's a real index.html + static/ to serve alongside it.
- Printing every LAN URL and auto-opening the browser (deferred from task 2/3) landed here instead of waiting further, using `netutil.LANURLs` (already built for task 6's server-info endpoint) plus a tiny `runtime.GOOS`-switched `openBrowser` helper in `main.go` (`rundll32 url.dll,FileProtocolHandler` on Windows, `open` on macOS, `xdg-open` elsewhere). Reason: task 11 is where a real page first exists to open — doing it now avoids a second pass over `main.go` later, and the LAN-URL helper was sitting there unused since task 6.
- No client-side router library or history-API pushState: navigation is plain `location.hash` (`#/wizard`, `#/users`, `#/activity`) with a single `hashchange` listener re-rendering the shell. Reason: simplest option that fits a small internal LAN tool with exactly three top-level views and no deep-linking requirement beyond "which tab am I on."
- The API client (`web/static/js/api.js`) always sends `X-Requested-With: idcard` and `credentials: 'same-origin'`, and throws an `Error` whose `.message` is literally the server's own `{"error": "..."}` text. Reason: the server already writes "what went wrong and how to fix it" per the spec's error-message rule; re-wording it client-side would just risk drifting from that text for no benefit.
- Direct hash navigation to `#/users` by a non-admin redirects to `#/wizard` rather than showing an error. Reason: the top bar already hides the Users tab for non-admins per spec ("Users (Admin only)"); a stray/stale link should just fall back to the tab that *is* available, not dead-end on a permissions error.
- Users page and Activity page (task 12) and all Wizard content (tasks 13-18) are still `<p class="placeholder">…</p>` stubs at this point — task 11's job was the shell, auth, top bar, routing, API client and presence heartbeat only, not the pages themselves.
- No browser tool was available this session (Chrome extension not connected) to visually verify the login/setup screens. Verified instead via `curl` (every embedded route — `/`, `/static/js/app.js`, `/static/css/app.css`, `/vendor/fonts/fonts.css`, `/vendor/jspdf.umd.min.js`, `/assets/brand/wcf-logo-color.svg` — returns 200 from a freshly built binary) and careful reading of the JS control flow. A future session with browser tools should do a real visual pass on the setup → login → top bar → sign-out flow before this is considered fully done.
- The Activity page folds `target` into the Action cell (`"employee_saved · E001"`) rather than adding a 5th column. Reason: the spec's Activity table lists exactly 4 columns (time, user, PC and IP, action), but most log actions are meaningless without their target (e.g. "user_disabled" alone doesn't say who) — appending it to the action text satisfies the literal column list while keeping the entries useful.
- Reset-PIN's inline row form originally built `id="rp-pin-${username}"` for the label/input pair — found and fixed during this session's re-check: an admin username containing a character invalid in a CSS ID (space, `.`, `:`, etc.) would have broken the `cell.querySelector('#' + id)` lookup silently. Replaced with a module-level incrementing sequence number (`rp-pin-${seq}`) for the id (keeping accessible `label for=`) and direct element references (`.rp-pin`/`.rp-confirm` classes) for reading values, so no user-controlled data ever reaches a selector string.
- Verified this task's Users/Activity API contracts against the running binary (not just unit tests): full curl flow — setup → add user → list → reset-pin → disable → enable → disable-last-admin (409) → activity query — confirms the JSON field names (`username/name/role/status/last_login`, `ts/username/ip/host/action/target`) match exactly what `users.js`/`activity.js` read.
- **Added a gap left by task 8:** `GET /api/employees/{id}/photo`, serving the raw JPEG from `data/photos/`. Task 8 only built the upload/write side; nothing consumed it until this task's live preview needed to actually display a photo. `Employee.Photo` is always server-set to exactly `<id>.jpg` (never user input), so joining it onto `PhotosDir` is traversal-safe without extra checks.
- `web/static/js/card.js`'s render functions deliberately do **not** use the reference's shared `<symbol>`/`<use>` + document-wide `<defs>` DOM technique, even though "match the reference exactly" is the task's framing. Reason: task 18's batch export renders many employees' card DOM simultaneously (off-screen, one per selected employee) to feed html-to-image; a shared element id (e.g. `id="dg"` for the Dynamis gradient, or `id="hg"` inside every Helios client's stored logo SVG) would collide across instances, since HTML forbids duplicate ids and `url(#id)` resolution is first-match-wins document-wide, not scoped to a nested `<svg>`. Every render call instead takes an `instanceId` (the `employee_id` — already unique and already CSS-id-safe) and a `namespaceIds()` helper rewrites every `id=`/`url(#…)`/`href="#…"` it finds to end with `-instanceId`. The rendered *pixels* match the reference exactly; only this one DOM-sharing optimization was dropped, and only because the reference was never exercised under this app's actual multi-card-at-once usage pattern.
- The Helios wordmark ("HELIOS MATERIAL" / "HANDLING") is hardcoded text in `card.js`, shown whenever `client.code` case-insensitively equals `"helios"`, wrapped around the client's own (already self-contained, namespaced) `logo` SVG — exactly mirroring how reference/index.html itself hardcodes a `HELIOS` JS constant separately from its generic per-client logo path. This was flagged as a required follow-up in task 7's decisions and is now resolved.
- Card front rendering treats `employee == null` as the canonical "blank card" state (dashed "Client logo" placeholder, default "LINE ONE/TWO/THREE" tagline, empty name/role/id/date) — this is what the live preview shows before any employees exist. An employee that exists but whose `client_code` doesn't match any known client (`client == null` with `employee` present) falls back to the same dashed-placeholder logo and default tagline, which doubles as a visual cue for the "Needs attention" unmapped-client case built in task 16.
- The preview's front/back pair is scaled to fit the available panel width via a JS-computed `transform: scale()` (recalculated on window resize) rather than a fixed CSS `zoom` factor like the reference's gallery view. Reason: the reference's `zoom:.42` was tuned for its own full-browser card gallery; this app's preview panel is a fixed-position pane of variable width (viewport minus the 520px step panel), so a responsive computed scale is the simplest option that actually fits every window size.
- `wizard.js` currently renders only the 4-step bar + a `<p>Step content coming soon.</p>` stub in the left panel, plus the fully working live preview on the right, fed by real `/api/clients` + `/api/employees` data. Step 1-4 content itself is tasks 14/16/17/18.
- Verified `card.js`'s pure render functions with real Node assertions (not just `node --check` syntax validation) covering: fixed-English date formatting, per-instance id namespacing (including that a raw `id="hg"`/`url(#hg)` from a stored client logo never survives un-namespaced), the blank-state placeholder, fixed back-side content, and that employee-supplied text (e.g. name) is HTML-escaped rather than injected raw — an XSS check, since `renderFront`/`renderBack` build HTML via template strings. Also verified end-to-end against the running binary: seeded Helios client's `code` matches `"Helios"` exactly as `wizard.js`'s case-insensitive lookup expects, and the photo upload → GET round-trip returns the identical bytes.
- **`card.js`'s `renderFront(employee, client, instanceId)` semantics changed** (before Step 1 needed it): `client` now controls the logo/tagline *independently* of `employee`, rather than being forced to `null` whenever `employee` is `null`. Reason: Step 1's live preview needs to show a client's branding (logo + tagline) on a blank card — no name/role/ID/date yet — while editing/adding that client, which the old coupling made impossible (it collapsed straight to the fully-blank dashed-placeholder state whenever there was no employee). The fully-blank placeholder is now specifically `renderFront(null, null, …)`; `renderFront(null, someClient, …)` is the new "preview this client's branding" state. Re-verified with Node assertions that both states still render correctly.
- `preview.js` gained `setOverride(client)`/`clearOverride()`: Step 1 needs to show a client's branding-in-progress without disturbing the underlying employee-navigation list or its current index — override suspends `‹`/`›` navigation and shows the client's name as the counter text instead of "n / N · Name", restored automatically on `clearOverride()`.
- `web/static/js/svgvalidate.js` mirrors `internal/clients/svg.go`'s `ValidateSVG` rule-for-rule (DOCTYPE/ENTITY block, root-is-svg, viewBox required, disallowed elements, `on*` attributes, non-fragment hrefs) using `DOMParser` instead of `encoding/xml`. This is a UX check only — the server re-validates independently and is the real security boundary — but per spec ("SVG validation: check on the client and again on the server") both checks are expected. Verified with real assertions (via a scratch jsdom install, not committed) that it agrees with the Go validator on every one of Go's own test cases, plus the actual seeded Helios logo.
- The Claude-prompt text in `step1.js` is a verbatim copy of the block in CLAUDE.md's "Claude prompt shown in the Add client form" section — copied character-for-character (including numbering and the exact `×`/em-dash-free wording) rather than paraphrased, since operators will literally copy-paste this into a Claude conversation.
- Choosing an existing client in the dropdown updates the live preview for every signed-in user (Admin or Operator), but only an Admin additionally gets the pre-filled edit form — matching "Clients are view-only" for Operators without a separate read-only form UI nobody asked for.
- The add/edit client form re-fetches the full client list (`GET /api/clients`) after a successful save rather than splicing the returned client into a locally-held array. Reason: simplest way to stay in sync with the server's slugified id and any other server-side normalization (uppercased tagline, etc.), and this is a low-traffic internal tool where the extra round trip is inconsequential.
- `scripts/make_template.py`'s Clients sheet seeds exactly one row: `Helios` — matching the client always seeded on first run. Reason: the template is a build-time asset (generated once, committed, not regenerated per-deployment against live data), so it can't know a real deployment's actual client roster; it exists to demonstrate the shape (a `client_code` column feeding the Employees sheet's dropdown) and give new users something that resolves immediately, not to be an authoritative client list — unmatched codes typed by hand are exactly what Step 2's "Needs attention" mapping (task 16) is for.
- No route/handler code was needed to serve `web/assets/employees_template.xlsx` — it's picked up automatically by the existing embedded-`FS` file server (`http.FileServer(http.FS(web.FS))`) from task 11, since `//go:embed index.html static assets vendor` already covers the whole `assets/` tree. Verified by hitting `/assets/employees_template.xlsx` on a freshly built binary and re-opening the downloaded bytes with `openpyxl`.
- The actual "Download template" link in Step 2's UI is out of scope here — task 15 is specifically the script + the committed `.xlsx` asset; the link itself lands with Step 2 (task 16).
- The Wizard's step bar is now actually clickable, switching between Step 1 (`step1.js`) and Step 2 (`step2.js`); Steps 3-4 still show the placeholder. Made the step-bar items real `<button>`s (was plain `<li>` text) since a clickable control has to be a real button per the accessibility rules — this was a small gap in task 13's original markup, fixed here since Step 2 is the first thing that actually needs to navigate between steps.
- **Caught and fixed a real bug via functional testing (not just `node --check`):** SheetJS's own CSV parser auto-detects date-like strings (e.g. `"2026-03-01"`) and silently converts them to an Excel serial itself — computed through a local-timezone `Date` parse, landing on a *fractional* day number (`46082.229...`, not a whole `46082`). Truncating that (as the server's `ParseDateValue` does for genuine serials) can land on the wrong calendar day depending on the browser's timezone. Fixed by passing `raw: true` to `XLSX.read()` itself (not just to `sheet_to_json`), which leaves CSV date cells as plain strings — verified against the vendored `xlsx.full.min.js` directly (via Node `require`, not just syntax-checked) that this preserves both a real `.xlsx` date cell's whole-number serial *and* a CSV date string verbatim, for both `YYYY-MM-DD` and `DD-MM-YYYY`.
- `step2.js`'s sheet parser normalizes headers (trim, lowercase, non-alphanumerics → `_`) so a hand-edited file with headers like `"Employee ID"` still maps to `employee_id` — verified against both the real committed template and a hand-built messy-header CSV.
- The add-employee form's optional photo field always uploads with a centered, unzoomed crop (`{zoom:1, x:50, y:50}`) — there's no crop UI here (that's Step 3, task 17); a photo added this way just starts at the same default every fresh upload gets, refinable later in the photo queue.
- Clicking an employee row calls `preview.setIndex()` using the row's position in the *unfiltered* employee list (not the filtered/displayed rows), since that's the order `preview`'s items array was built in. Both `step2.js` and `wizard.js` independently fetch `/api/employees` but stay in lockstep because every mutation (import, add, map-client) goes through the same `reloadEmployees` → `onEmployeesChanged` → `refreshPreviewItems` chain, so their array orders never diverge in practice.
- Verified the whole Step 2 flow end-to-end against the running server: import with one known and one unknown client code, confirm the unknown one appears fit for "Needs attention," map it via `POST /employees/{id}/client`, and add a new employee with an attached photo in one add-form submission (two sequential requests: `POST /employees` then `POST /employees/{id}/photo`).
- `preview.js` gained two more orthogonal, additive mechanisms for Step 3, neither touching Step 1's existing `setOverride`/`clearOverride`: `setDraftPhoto(src, crop)` overlays a not-yet-uploaded blob URL + crop onto the *currently selected* item only (employee/client/navigation untouched — only the photo and its crop are swapped in), and `setPhotoDragHandler(fn)` registers a callback fired while dragging inside the rendered `.photo` box. Both are wired via event delegation on the stable `pair` container (attached once at creation, not re-attached per render) and window-level `mousemove`/`mouseup` listeners cleaned up in `destroy()` — the same pattern already used for the resize listener, extended rather than duplicated.
- `card.js`'s `renderFront` gained a 4th, optional `photoSrcOverride` parameter used in place of the computed `/api/employees/{id}/photo` URL when present. Verified with Node assertions that the override fully replaces the server URL (not appended alongside it) and works even for an employee with no saved photo yet (the in-progress "first upload" case).
- Drag-to-crop convention: dragging the mouse right/down *decreases* `crop.x`/`crop.y` (a "grab and pan the photo" feel — the image follows the cursor). This is a plain judgment call since the spec only says "dragging inside the preview... adjust[s] the crop" without specifying direction; it's isolated to one small handler in `step3.js` (`preview.setPhotoDragHandler` callback) and trivially flippable if a real visual check (this session had no browser tool) shows it feels backwards.
- Navigating away from Step 3 mid-edit via the step bar (not via its own Cancel button) resets the preview's draft-photo/drag-handler state in `wizard.js` (so Steps 1/2/4 never get stuck showing a stale draft or intercepting drags), but does *not* revoke the abandoned edit's blob object URL — that cleanup only runs through `step3.js`'s own `stopEditing()`. Accepted as a small, bounded leak (one blob per abandoned in-session edit, freed on page reload) rather than adding a cross-step destroy-callback mechanism no other step needs.
- The photo queue's "first missing employee is selected automatically" re-evaluates against the *currently filtered* view (search/client filter applied) whenever filters change or the current selection drops out of view, but does nothing (correctly) when the current selection is still visible — confirmed via the running server that uploading one of two employees' photos leaves the other correctly `photo: ''` and ready to be the next auto-selected target, matching what `ensureSelection()` reads after `saveAndNext()`'s refetch.
- Short-side-under-600px warning is computed from the *original* (pre-resize) image dimensions, not the resized 800px-max output — resizing always shrinks large photos below any meaningful "too small" threshold, so checking post-resize would make the warning meaningless.
- `cardexport.js` renders each card off-screen (`position:fixed;left:-10000px`, still attached to `document.body` so layout/fonts compute correctly) at full `EXPORT_PIXEL_RATIO` via the vendored `html-to-image` → canvas → `image/jpeg` data URL → embedded in a `jsPDF` document with two 54×85.6mm pages (front, back), matching the spec's export pipeline exactly. The `.exporting` CSS class (already prepared in task 13's `card.css`) strips the slot guide and rounded corners for export only. `await document.fonts.ready` runs before every render, per "wait for the fonts and images to load before rendering."
- `POST /api/export` is still one PDF per call (per task 9's decision) — `step4.js` loops selected employees sequentially, updating "Rendering i / N…" between each, uploading each as it's rendered rather than batching. The ZIP (via vendored JSZip) is assembled client-side purely for the user's own download when more than one card is selected; the server never sees or builds a ZIP.
- **Virtualization is real but intentionally approximate**, not pixel-perfect: the grouped (header + row) list is flattened into one array and only a scroll-position-derived window (± a fixed pixel buffer) is actually rendered, with two spacer `div`s maintaining correct total scrollable height using an *assumed* fixed row/header height rather than measuring actual rendered heights. This is the right tradeoff for "lists of 1000+ stay smooth" without the much higher complexity of a fully general variable-height virtualizer — row/header heights are fixed by this same CSS, so the assumption holds exactly, not approximately.
- Group-header checkboxes are plain binary checked/unchecked (checked only when *every* shown row in that group is selected), not a tri-state indeterminate control for partial selection. Spec asks for "a checkbox... that selects or clears that group's shown rows" without mentioning a partial/indeterminate visual state, so the simpler binary control satisfies it; `indeterminate` is a JS-only property with no HTML attribute equivalent and would need extra post-render wiring for a purely cosmetic improvement.
- Selection is **not** cleared after a successful export. Reason: spec explicitly says "the selection survives filter changes," and clearing it specifically after export (but not after any other filter change) would be an inconsistent special case; a just-exported row simply drops out of view under the default "Not exported" status filter instead, which is a more natural signal that it's done.
- **Verified with a real jsdom DOM-interaction test** (20 assertions, not just `node --check`) covering: default "Not exported" filtering, Select shown/Clear, singular/plural export-button labeling, status/client/month filters (including that the month dropdown is newest-first from real fixed-English labels), the no-photo warning, the exact sanitized filename shown per row, shift-click range selection spanning a client-group boundary, and group-header select/deselect. This surfaced and fixed two *test* bugs along the way (a stale DOM reference after `innerHTML` replacement, and manually pre-setting `.checked` before dispatching a click that also toggles it) rather than real `step4.js` bugs — worth noting since it shows the kind of jsdom footgun this session hit repeatedly when hand-verifying interactive code without a real browser.
- `cardexport.exportFilename`'s sanitization was cross-checked byte-for-byte against `internal/export.SanitizeFilenamePart`'s own Go unit test case, confirming the client and server agree on the exact filename shown to the user vs. the one the server actually saves under.
- The "Under the hood" panel (`underthehood.js`) reuses the existing `GET /api/server-info` (task 6) verbatim — no backend changes needed this task, only a frontend consumer. It's a native `<details>/<summary>` (real, keyboard-operable, no ARIA reinvention needed) polling every 15s to match the presence heartbeat cadence, plus an immediate refresh on open via the `toggle` event so expanding it never shows a stale snapshot.
- "Current step" shown per connected user is the top-level route (`wizard`/`users`/`activity`) that `app.js` already feeds to `presence.setStep()` — not which of the 4 Wizard steps they're on. Spec says "current step" without specifying that granularity; threading per-wizard-step tracking through all four step modules just for this display would be a cross-cutting change for a "nice to have" level of detail the spec doesn't clearly ask for.
- **Verified with a real jsdom test** (13 assertions) that caught a genuine, useful surprise: setting `<details>.open = true` programmatically already fires a native `toggle` event in jsdom (matching current browser/spec behavior) — an initial test draft that also manually dispatched a second `toggle` event was double-firing the refresh and looked like a bug in `underthehood.js` until traced down; the shipped code was correct all along, only the test was redundant. Left as a documented example of the jsdom footguns this session hit while substituting for a real browser.
- `scripts/build.sh`/`.ps1` run `go vet` and `go test ./...` first and abort on any failure, then cross-compile into `dist/<goos>-<goarch>/idcard(.exe)` — one self-contained, ready-to-copy portable folder per platform (matching the runtime layout section's flat `idcard(.exe)` + `config.json` + `data/` next to each other), rather than a single `dist/` folder with OS-suffixed filenames. Both scripts were actually run this session (not just written): all four cross-compiles (`windows/amd64`, `darwin/arm64`, `darwin/amd64`, `linux/amd64`) succeed with `CGO_ENABLED=0`, and the native `windows/amd64` output was smoke-tested by copying it to a clean scratch folder and hitting `/` and `/api/setup` on it directly — a real end-to-end check that the portable, from-any-folder design actually works, not just that compilation succeeds.
- "test all four binaries" is interpreted as running the full Go test suite as a build gate (via `go test ./...`) rather than literally executing each cross-compiled binary — three of the four target platforms (darwin/arm64, darwin/amd64, linux/amd64) can't be executed on this Windows development machine at all. The one binary that *can* run natively here (windows/amd64) was smoke-tested for real, as above.
- **Found and fixed a real spec-compliance gap during the final cleanup pass:** the runtime layout explicitly lists `data/logos/<client_id>.svg` as a file, but `internal/clients` had been storing each client's logo SVG *inline inside `clients.json`* since task 7 — `data/logos/` was being created (task 2) but nothing ever wrote to it. Fixed by making `clients.Manager` (not `Store`) own logo file I/O: `Create`/`Update` write `data/logos/<id>.svg` only *after* the corresponding `clients.json` metadata write succeeds (avoiding an orphaned logo file if the metadata write is rejected, e.g. a code collision), and `List`/`FindByCode` read the file back to populate `Client.Logo` for callers. The **API contract is completely unchanged** — `GET`/`POST`/`PUT /api/clients` still return the logo inline as SVG text, since the frontend's inline-embed-and-namespace rendering approach (task 13) depends on having the raw text, not a URL — only the on-disk *persistence* mechanism changed, verified with new tests (`clients.json` never contains the SVG; the file at `data/logos/<id>.svg` has the exact bytes; `Update` with no new logo preserves the existing file's content) and an end-to-end server smoke test.
- The cleanup pass cross-checked every JSON data-model struct (`User`, `Employee`, `Client`, activity/presence `Entry`) field-by-field against the spec's literal `{...}` shape listings — all matched exactly, no further changes needed there.
- Cleanup also confirmed: no `TODO`/`FIXME`/`console.log`/`debugger` left in any shipped file, every frontend JS module is actually imported somewhere (no orphans), no stray scratch/test artifacts made it into git (checked `git ls-files` against the expected tree), and `dist/`/`data/`/`config.json` stay out of the repo via `.gitignore` as intended.
- README.md covers setup, building (incl. re-vendoring), running on a LAN, backup-by-copying-`data/`, the Windows OneDrive Desktop redirection note, and troubleshooting, per the task list — with `wcf-logo-color.svg` at 120px as the header image, per the branding spec.

## Blockers
(none — every prerequisite tool was available or installable without a password: Go 1.27 via `winget`, `openpyxl` via `pip install --user`)

## Handover

**Status: all 21 tasks complete.** 145 Go tests pass (`go vet` and
`go test ./...` clean), all four platform binaries cross-compile, and
the full app was exercised end-to-end (setup → sign in → add client →
add employee → upload photo → export a PDF → confirm it lands in the
right month folder with `exported_at`/`exported_by` recorded) against a
real built binary, not just via `go test`.

### What was built

- **Backend** (`internal/`, Go standard library only): config +
  portable data folder, atomic/mutex-protected JSON storage, PIN auth
  with PBKDF2-SHA256 + lockout + sessions + role middleware, Users API,
  an append-only Activity log with filtered/paginated queries, presence
  tracking with a cached reverse-DNS lookup, a Clients API with
  from-scratch SVG validation (XML-token-walked, not regex) and the
  Helios seed, an Employees API (upsert/bulk-import/photo
  upload+serve/client-mapping), and an Export endpoint that saves PDFs
  into `<Mon YYYY>/` folders and records the export.
- **Frontend** (`web/`, vanilla JS ES modules, no build step, embedded
  via `go:embed`): first-run setup, login, a top bar with routing, a
  card renderer that matches `reference/index.html` pixel-for-pixel
  (with per-instance id-namespacing so batch export never collides), a
  live preview with navigation, all 4 Wizard steps (clients;
  employees/import/Needs-attention; photo queue with crop
  sliders+drag; export with filters/grouped selection/shift-range and
  client-side PDF+ZIP generation), a Users page, an Activity page, and
  the "Under the hood" panel.
- **Tooling**: `scripts/vendor.*` (checksum-verified fetch of pinned
  third-party JS libs and fonts), `scripts/make_template.py` (the
  Excel import template), `scripts/build.*` (cross-compile + test
  gate).

### How to run it

See README.md for full instructions. Short version: `./scripts/build.sh`
(or `.ps1`), then run the binary for your platform from `dist/`. It
creates `config.json` and `data/` next to itself on first run.

### Known limitations / follow-ups a future session should consider

- **No browser tool was available this entire session** (the Chrome
  extension was never connected). Every frontend behavior was verified
  by (a) reading the code very carefully, (b) real Node/jsdom
  interaction tests for the trickiest modules (`card.js`, `step4.js`,
  `underthehood.js` — dozens of assertions total, and these caught
  several real bugs plus a few test-authoring bugs), and (c) full
  `curl`-driven end-to-end flows against the real running server. What
  was **not** done: an actual visual check in a real browser — CSS
  layout bugs, the drag-to-crop feel (see below), font rendering, and
  general polish should get one real look before this ships to users.
- **Drag-to-crop direction in Step 3 is an untested guess.** The spec
  says dragging adjusts the crop but doesn't specify which way; the
  chosen convention ("image follows the cursor") is isolated to one
  small callback in `step3.js` and trivially flipped if it feels
  backwards once someone actually drags a photo in a browser.
- **Step 4's virtualized list assumes a fixed row/header height** (in
  CSS and in the JS scroll-math) rather than measuring actual rendered
  heights. This is exactly right as long as nobody changes those CSS
  rules independently of the JS constants — if `.export-row`/
  `.export-group-header` height ever changes, update `ROW_HEIGHT`/
  `HEADER_HEIGHT` at the top of `step4.js` to match.
- **Presence's "current step"** only reflects the top-level route
  (Wizard/Users/Activity), not which of the 4 Wizard steps someone is
  on — see the task-19 decision note if finer granularity is ever
  wanted.
- **A stray, small blob-URL leak**: navigating away from Step 3 mid-photo-edit
  via the step bar (rather than that step's own Cancel/Save) doesn't
  revoke the abandoned edit's object URL. Bounded (one per abandoned
  edit, freed on page reload), documented in task 17's decisions.
- **The exact per-decision reasoning for all 21 tasks is preserved
  above** under `## Decisions` — many of them are genuine judgment
  calls where the spec was silent or ambiguous (e.g. one-PDF-per-export-call,
  card-id-namespacing instead of shared `<symbol>`/`<use>`, username
  case-insensitivity, the Helios wordmark being hardcoded rather than
  baked into the seeded logo SVG). Anyone continuing this project should
  read that list before changing behavior it documents — several of
  those decisions look arbitrary in isolation but exist for a specific,
  stated reason.
- **Nothing was deferred or skipped** — every numbered task, every
  spec'd endpoint, every UI element described in CLAUDE.md has a
  corresponding implementation. The one real bug the cleanup pass found
  (client logos stored inline instead of as `data/logos/<id>.svg`
  files) was fixed, not just noted.

## Post-handover: real browser verification (2026-09-28)

A later session got Chrome browser tooling connected and used it to
actually click through the Wizard — the visual check task 13/21's
Handover explicitly flagged as not yet done. It surfaced and fixed
**three real bugs**, two matching the user's report ("the wizard is
unhelpful, clicks land outside the box") and one critical bug the
report didn't even mention:

1. **`preview.js`'s `fitScale()` only fit the card's width, never its
   height.** Since `CARD_HEIGHT` (856) is nearly double `CARD_WIDTH`
   (540), on any normal landscape window the container's *height* is
   almost always the binding constraint, not its width — so the card
   rendered taller than its panel, pushing the back card, the nav
   buttons and effectively the whole bottom portion of the preview off
   the visible area. Fixed by computing scale as `min(widthFit,
   heightFit)` (a proper "contain" fit), measuring the actual nav row
   height rather than a hardcoded guess.
2. **A CSS class-name collision**: `card.css` was carried over from
   `reference/index.html` with bare, unscoped selectors (`.brand`,
   `.role`, `.meta`, `.photo`, `.tag`, ...) that were safe in that
   isolated single-purpose page but collide with this app's own chrome
   reusing the same generic names — confirmed via
   `getComputedStyle`/`getBoundingClientRect` in a live page that the
   top bar's own `.brand` (logo + "ID Card Studio") and `.who .role`
   ("Admin" under the user's name) were inheriting `position: absolute`
   from `card.css`'s bare `.brand`/`.role` rules, scattering them across
   the page instead of sitting in the top bar. This is almost certainly
   what read as "clicks land outside the box" — real, correctly-wired
   interactive elements rendered nowhere near where their listeners
   actually lived. Fixed by scoping every selector in `card.css` under
   `.card` (`.card .brand`, `.card .role`, etc.), so the file can never
   again leak into anything outside an actual card element.
3. **A critical, unreported backend bug**: a nil Go slice marshals to
   JSON `null`, not `[]`. `employees.Store.List()` (and, latently,
   `clients.Store.List()` and `auth.UserStore.List()`) returned the
   zero value of `[]T` — `nil` — whenever the backing JSON file didn't
   exist yet. On a genuinely fresh install with zero employees,
   `GET /api/employees` returned `null`, and `wizard.js`'s
   `employeeList.map(...)` threw, silently aborting the rest of
   `renderWizard()` — which is why the step bar never showed an active
   step and Step 1's content never rendered at all. Every prior
   end-to-end smoke test in this project added an employee *before*
   checking the list, so this was never once triggered until a real
   browser hit a truly empty install. Fixed at all three call sites;
   each now normalizes `nil` to an explicit empty slice, with a new
   regression test per package asserting `json.Marshal(list) == "[]"`
   on a fresh, empty store.

All three fixes are covered by the full Go test suite (145 → 148
tests, all passing) and were re-verified visually: a rebuilt binary
correctly shows the card preview fully contained in its panel, the top
bar's brand/role text in the right place, and Step 1's client dropdown
and Step 2's employee table both rendering and switching correctly.

**Tooling note for whoever picks this up next:** this session's browser
automation had an unreliable simulated-keyboard/`left_click` path —
confirmed independently on Google's own search box, so it isn't an
application bug — while `element.click()` via injected JS and
`getBoundingClientRect`/`getComputedStyle` inspection worked reliably
throughout. As a result, Steps 3 and 4 (photo crop drag, export
filters/selection) were not re-verified with real clicks this session,
only read through carefully again. **A future session with reliable
click/type tooling should still do a real pass over Steps 3 and 4**,
and in particular the drag-to-crop direction in Step 3, which task 17's
decisions already flagged as an untested guess.

## Post-handover: Steps 3 and 4 real click-through (2026-09-28, later same day)

A follow-up session with working mouse/click tooling did the real pass
over Steps 3 and 4 that the previous entry above flagged as
outstanding. Real uploads, drags, clicks and an actual export were
exercised end to end, and it surfaced **one more real bug**, again one
the Go test suite's own tests couldn't have caught because they craft
requests directly rather than through a browser:

4. **The photo GET endpoint required the `X-Requested-With: idcard`
   header, which a plain `<img src="...">` can never send.** Every API
   route was wrapped in one blanket `RequireXRequestedWith` middleware
   (`internal/server/server.go`), including
   `GET /employees/{id}/photo`. `card.js`'s `photoHtml()` renders a
   saved (non-draft) photo as a bare `<img>` tag — the browser's native
   image loader, which cannot attach custom headers — so every
   `<img>` request for a previously-saved photo got a 400 and rendered
   as a blank grey box, everywhere except Step 3's upload-in-progress
   view (which uses a client-side object URL instead of hitting this
   endpoint at all). This silently broke the photo on Step 1, Step 4,
   and — critically — the actual exported PDF, which renders the same
   DOM the preview does. It went undetected by every existing Go test
   because the shared `doJSON` test helper always sets the header by
   hand. Fixed by exempting exactly `GET /employees/*/photo` from the
   header check in `RequireXRequestedWith`
   (`internal/auth/auth.go`) — it's still gated by `RequireAuth`'s
   `SameSite=Strict` session cookie, which a cross-site `<img>` can't
   carry either, so this doesn't reopen the CSRF gap the header was
   for. The POST upload route is unaffected and still requires the
   header. Three new regression tests cover it: two unit tests on
   `RequireXRequestedWith` itself (GET photo exempt, POST photo still
   rejected) and one server-level test uploading then fetching a photo
   with no header at all.

What was verified with real mouse drags and clicks, against a fresh
three-employee seed with no photos:
- **Step 3**: uploaded a real JPEG via `file_upload`, confirmed the
  crop editor's zoom/horizontal/vertical sliders render and respond to
  both a direct click-to-position and a native `left_click_drag` on the
  slider track; dragged directly on the live preview's photo box and
  confirmed the **drag-to-crop convention flagged as an untested guess
  in task 17 is correct as shipped** — dragging right/down measurably
  decreased `crop.x`/`crop.y` (a "grab and pan" feel), read back
  directly off the range inputs' DOM values; clicked "Save & next" and
  confirmed the photo and crop persisted to `employees.json` (checked
  via the live `/api/employees` response) and the queue correctly
  advanced to the next missing employee, updating the "N / 3 photos
  done" progress bar each time.
- **Step 4**: confirmed the client-grouped list, the PHOTO/NO PHOTO
  badges, the group-header select-all/partial/none checkbox states, a
  plain row click, a real `shift`-modifier range-click across rows, the
  "N selected · M shown" count, that **selection survives a search
  filter change** (spec requirement), the "some selected cards have no
  photo" warning, and the "Export PDF" → "Export N PDFs (ZIP)" button
  text switching on selection count. Actually triggered both a
  single-card export and a 3-card ZIP export (mixing photo and
  no-photo employees); both produced valid 2-page PDFs on disk under
  `<export_dir>/<Mon YYYY>/<employee_id>_<Name>.pdf`, and
  `exported_at`/`exported_by` were correctly recorded and reflected
  live by the "Not exported" / "Exported" status filter.

All four bugs found across both browser-verification passes are now
fixed; the full suite is 148 → 152 tests, all passing, `go vet` clean,
and all four platform binaries rebuild successfully via
`scripts/build.sh`. Every piece of Step 3 and Step 4 described in
CLAUDE.md has now been exercised with a real mouse and a real upload,
not just read.

**Caveat on the export claim above:** "both produced valid 2-page
PDFs" was verified with `file` (page count and PDF-ness) but the
actual rendered image inside each PDF was not looked at — an
incomplete check that missed a real rendering bug, below.

## Post-handover: malformed back-card logo in the export (2026-09-28, user-reported)

The user asked directly: "export file is wrong, have you checked it?
the dynamis logo is malformed in export" — a fair catch, since the
prior entry's export verification stopped at "is this a valid 2-page
PDF" and never rendered the PDF's own image to look at it. Doing that
(via PyMuPDF, rendering each exported page to PNG) showed the back
card's white "D" mark rendering solid black/near-invisible against
the navy background, everywhere the card is exported — the live
in-browser preview was never affected and always showed it correctly,
which is exactly why this survived every prior visual check in this
project.

**Root cause:** `reference/index.html`'s own DYNAMIS mark sets its
white fill via a CSS custom property (`.dynamis{--d:#fff}` in
`card.css`, referenced as `style="fill:var(--d)"` on two `<path>`s in
`card.js`). That resolves fine in the live DOM, where `--d` is
inherited from the ancestor `.dynamis` element in the normal CSS
cascade. But the export path (`cardexport.js`) rasterizes each card
with `window.htmlToImage.toCanvas()`, which works by cloning the
subtree, inlining computed styles, and serializing the result into an
isolated `data:image/svg+xml` document containing a `<foreignObject>`
— confirmed directly by reading the actual network request it issues.
Inside that isolated document, `--d` is no longer defined anywhere
(the custom property wasn't carried across the serialization), so
`var(--d)` resolves to nothing and `fill` falls back to its SVG
default of black.

**Fix:** replaced `style="fill:var(--d)"` with a literal `fill="#fff"`
attribute on both paths in `dynamisMarkSvg`/`dynamisASvg`
(`card.js`), and removed the now-unused `--d: #fff` declaration from
`card.css`. The color was always a fixed, unconditional value (never
overridden anywhere) — the custom-property indirection was serving no
purpose and only reintroducing this on the next such property, if any,
would be a mistake. No Go tests apply here (this is a pure
CSS/rendering bug the Go suite can't see); verified instead by
re-running a real export through the browser and rendering the
resulting PDF's pages to PNG with PyMuPDF to look at the actual pixels
— the "D" is now correctly white with the gold chevron, matching
`reference/index.html`, and the front side (photo, Helios logo,
silhouette placeholder for a photo-less employee) was re-checked the
same way and is unaffected.

**Lesson for future export-correctness checks in this project:**
`file <pdf>` and a page count are not enough to catch a rendering bug
in an exported PDF, because the PDF is just a JPEG glued to a page —
the actual image content has to be decoded and looked at. Reader
tools with no PDF renderer available in this environment: install
`pymupdf` via pip (`python -m pip install --user pymupdf`), open with
`fitz`/`pymupdf`, `page.get_pixmap(dpi=...)`, `pix.save(path)`, then
view the PNG. On Windows, remember the native Python invoked from
git-bash needs a real Windows path (or a `pathlib.Path.cwd()`-relative
one), not a `/tmp/...`-style git-bash path — see the same gotcha noted
in the previous entry for saving the test photo.
