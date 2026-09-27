# ID Card Studio — White Coat Foundry

A portable, LAN-hosted ID card generator. Staff at Dynamis (a placement company) make photo ID cards for employees placed at client companies (e.g. Helios Material Handling, Anulekha Hospital) and export print-ready PDFs.

## Autonomous operation (read first, every session)

You work unattended across multiple sessions. Sessions may end at any moment when usage limits hit, and a new session resumes later.

1. **Start of every session:**
   - Read `PROGRESS.md`.
   - Run `git status` and `git log --oneline -15`.
   - Continue from the first unchecked task.
   - If `PROGRESS.md` doesn't exist, create it from the task list at the end of this file, then begin.
2. **Work in small units.** After each task: build, `go vet`, run the tests, commit, then tick the task in `PROGRESS.md` in the same commit. Never leave more than one task's work uncommitted.
3. **Commits:** Conventional Commits (`feat:`, `fix:`, `test:`, `docs:`, `chore:`), one task per commit. The repo must always build.
4. **Interrupted work:** if you find uncommitted changes at the start of a session, check them against the current task. Finish and commit them, or `git stash` them with a note in `PROGRESS.md`.
5. **Decisions:** when the spec is silent, choose the simplest option that fits the spec. Record it under `## Decisions` in `PROGRESS.md`, with one line on the choice and one on the reason, and keep going. Don't stop to ask.
6. **Blockers:** if something truly blocks you (e.g. a missing tool that can't be installed without a password), log it under `## Blockers` with what's needed, skip to the next task that doesn't depend on it, and continue.
7. **Prerequisites:** check for Go 1.24+, git, Python 3 with `openpyxl`, and curl or PowerShell.
   - Install anything missing without prompting if you can (`pip install --user openpyxl`, or a package manager that needs no sudo).
   - Otherwise treat it as a blocker (rule 6).
8. **Finish:** when every task is ticked, run the full test suite and a cross-compile, update the README, and write a `## Handover` section in `PROGRESS.md` covering what was built, how to run it, known limitations and follow-ups.

## Code rules
- Keep the code lean and clean: no dead code, unused dependencies, commented-out blocks or speculative features.
- The only zip functionality is the app's own export ZIP feature. Never package the project itself as a zip.
- Brand the UI "ID Card Studio · White Coat Foundry".
- UI copy is in sentence case: plain verbs, and buttons say exactly what they do. Error messages say what went wrong and how to fix it.
- Accessibility:
  - use real `<button>`, `<a>`, `<label>` and `<input>` elements
  - show a visible keyboard focus
  - make touch targets at least 44 px

## Reference
`reference/index.html` holds the approved card templates. If it doesn't exist yet, create it from **Appendix A** at the end of this file, byte for byte. Match it exactly: logo SVGs (Helios sunburst, Dynamis D and wordmark), layout values, text and colours. Screenshots of the approved wizard mockup may also be in `reference/`.

## Stack
- **Server:** Go 1.24+, standard library only (no third-party modules).
  - Cross-compile windows/amd64, darwin/arm64, darwin/amd64 and linux/amd64 into `dist/`.
- **Frontend:** vanilla JS (ES modules), HTML and CSS, with no framework and no build step. Embed it in the binary with `go:embed`.
- **Vendored in `web/vendor/`**, pinned, no CDN at runtime: html-to-image, jsPDF, SheetJS (xlsx), JSZip, and Montserrat plus Josefin Sans as woff2 (OFL-licensed).
- **Scripts:**
  - `scripts/vendor.sh` and `.ps1` fetch the pinned libraries and fonts
  - `scripts/build.sh` and `.ps1` cross-compile
  - `scripts/make_template.py` generates the Excel template
- **Commit to the repo:** the vendored files and the generated `web/assets/employees_template.xlsx`, so a rebuild needs only Go.

## Runtime layout (portable: runs from any folder or USB drive)
```
ID-Card-Studio/
├── idcard(.exe)
├── config.json   {"port":8080,"export_dir":"~/Desktop/ID Cards","open_browser":true}
└── data/
    ├── clients.json, employees.json, users.json, activity.jsonl
    ├── logos/<client_id>.svg
    └── photos/<employee_id>.jpg
```
- Create missing files and folders on first run. Expand `~` to the user's home folder.
- On start, print every LAN URL and open the host's browser.
- Bind to 0.0.0.0.
- Every write is atomic (temp file, then rename) and protected by a mutex, because several users work at once.

## Data model
- **Client:** `{id, name, code, logo, tagline:[3 uppercase strings]}`
  - `code` is the value used in the Excel `client` column. Matching is case-insensitive.
  - Seed Helios Material Handling on first run: code `Helios`, the built-in SVG from the reference, tagline `ASSETS / PEOPLE / PERFORMANCE`.
- **Employee:** `{employee_id (unique), name, role, client_code, join_date (YYYY-MM-DD), photo_note, photo, crop:{zoom,x,y}, updated_by, updated_at, exported_at, exported_by}`
- **User:** `{username, name, role: "admin"|"operator", pin_hash, salt, disabled, last_login, failed_attempts, locked_until}`
- **Activity** (one JSON object per line): `{ts, username, ip, host, action, target}`

## Authentication and users
- **First run:** if no users exist, show a setup screen to create the first Admin.
- **Login:** username plus a 4–6 digit PIN.
  - Hash PINs with PBKDF2-SHA256 (`crypto/pbkdf2`), a random 16-byte salt and at least 210,000 iterations.
  - After 5 failed attempts, lock the account for 5 minutes.
- **Session:** a random 32-byte token in an HttpOnly, SameSite=Strict cookie, lasting 12 hours.
  - Every API call must send the `X-Requested-With: idcard` header.
- **Roles:**
  - Admin: everything.
  - Operator: employees, photos and export. Clients are view-only. No Users page.
- **Users page (Admin only):**
  - Table: name, username, role, status, last login, with Reset PIN and Disable/Enable actions.
  - Add-user form: name, username, role, PIN, confirm PIN.
  - Users are never deleted. The last active admin can't be disabled or demoted.
- **Activity page:**
  - Table: time, user, PC and IP, action.
  - Filters for user and date range, plus a text search, with the newest entries first and paginated.
  - Log sign-ins, failed sign-ins, user changes, client, employee and photo changes, imports and exports.

## Presence (the "Under the hood" panel)
- Each browser tab sends `POST /api/presence {step}` every 15 seconds.
- The server keeps connected users in memory: username, IP, and a hostname from `net.LookupAddr`, cached for 10 minutes and falling back to the IP.
- A user is dropped after 60 seconds of silence.
- The collapsible panel at the bottom of the wizard's left panel shows:
  - "N users connected"
  - one row per user: username (marked "(you)" for the viewer), hostname · IP, current step, last seen
  - server details: URL, uptime, data folder, export folder

## UI layout
- **Top bar** (charcoal): the app name, the Wizard | Users (Admin only) | Activity tabs with a coral underline on the active tab, the user's name and role, and Sign out.
- **Wizard:**
  - Left panel, 520 px wide, background `#F4F4F2`: a 4-step bar at the top, then the current step's content. The panel scrolls.
  - Right side (`#2E3238`): a live preview of the front and back, with ‹ › arrows and an "n / N · Name" counter.

### Step 1 · Client
- Before a client is chosen, the preview shows a blank card with a dashed "Client logo" placeholder and the tagline "LINE ONE / LINE TWO / LINE THREE".
- A client dropdown sits next to a **+ Add client** button (Admin only). Choosing a client loads its assets into the preview and into an edit form.
- The Add client form has two parts:
  1. **"Prepare the logo with Claude":** the SVG prompt below in a read-only box with a Copy prompt button.
  2. **"Client details":** name, Excel client code, logo (upload an SVG file or paste SVG code), and three tagline lines. The preview updates live.
- **SVG validation:** check on the client and again on the server. Reject an SVG if it:
  - is not SVG
  - contains `<text>`
  - has no `viewBox`
  - contains `<image>`, `<script>`, `<foreignObject>`, `on*` attributes, or external `href`s
- Store clean SVG only, and fit the logo inside the logo box.

### Step 2 · Employees
- Upload `.xlsx` or `.csv` using SheetJS, then upsert rows by `employee_id`.
  - Columns: `employee_id, name, role, client, join_date, photo_note`
  - Accept Excel date serial numbers, `YYYY-MM-DD` and `DD-MM-YYYY`.
  - Report skipped rows along with the reason for each.
- Provide a **Download template** link to `/assets/employees_template.xlsx`, containing:
  - an **Employees** sheet with a styled header, one example row and a date-formatted `join_date` column
  - a **Clients** sheet listing client codes
  - a data-validation dropdown on the `client` column that reads from the Clients sheet
- Show a table with ID, name and client columns (fixed layout, long text cut off with an ellipsis). Clicking a row selects it for the preview.
- Rows with an unknown client code appear in a "Needs attention" block, with a dropdown to map each one to an existing client.
- An **+ Add employee** form: ID, join date, name, role, client, and an optional photo upload.
- Search by name or ID, plus a client filter.

### Step 3 · Photos
- **No bulk upload.**
- Show a queue with missing photos at the top and done ones below. The first missing employee is selected automatically.
- **Current employee box:**
  - name, ID, and the Excel photo note shown as a hint only
  - an **Upload photo** button, which becomes **Replace photo** once a photo exists
- **After an upload:**
  - The preview updates at once.
  - Zoom (1–2.5), horizontal and vertical sliders, plus dragging inside the preview, adjust the crop.
  - Show a warning if the photo's short side is under 600 px.
  - **Save & next** (Enter also works) and **Cancel**, which keeps the old photo.
- On the client, resize photos to a maximum of 800 px on the long side, as JPEG at quality 0.9, before uploading.
- Show a progress bar: "x / N photos done".
- Search by name or ID, plus a client filter.

### Step 4 · Export
- **Filters:**
  - search by name or ID
  - client
  - month of the join date ("Sep 2026", newest first, built from the data)
  - export status: Not exported (default), Exported, Any
- **Selection:**
  - The list is grouped by client, with a checkbox on each group header that selects or clears that group's shown rows.
  - Each row has a checkbox, the file name `<employee_id>_<Name>.pdf` (characters outside `[A-Za-z0-9_-]` become `_`), "Joined DD Mon YYYY · exported DD Mon YYYY by user" (or "not exported"), and a Photo or No photo badge.
  - Shift-click selects a range.
  - **Select shown** and **Clear** buttons.
  - Show "N selected · M shown". The selection survives filter changes.
  - Render only the visible rows so lists of 1000+ stay smooth.
- If any selected card has no photo, show a warning. Export is still allowed, and those cards use the silhouette.
- **Export button:**
  - With one card selected it reads "Export PDF"; with several, "Export N PDFs (ZIP)".
  - The browser renders each card and builds the PDFs.
  - It POSTs them to `/api/export`, and the server saves them to `<export_dir>/<Mon YYYY>/<file>.pdf`.
  - The same file (single card) or a JSZip archive (several cards) is also downloaded on the user's PC.
  - The server records `exported_at` and `exported_by` and logs the export.
  - Show progress ("Rendering 12 / 40…").
- Use a fixed English month list (Jan…Dec) everywhere, so the output is the same on every computer.

## App branding (White Coat Foundry)
- Use the official White Coat Foundry badge logos from Appendix B, saved exactly as given in `web/assets/brand/`:
  - `wcf-logo-color.svg`: full colour (black ink, orange accents), for light backgrounds
  - `wcf-logo-subtle-dark.svg`: muted grey, for dark backgrounds
- **Placement:**
  - Sign-in and first-run setup screens: `wcf-logo-color.svg` at 96 px, centred above "ID Card Studio".
  - Top bar: `wcf-logo-subtle-dark.svg` at 36 px, left of the app name.
  - Favicon: `wcf-logo-color.svg`.
  - README: `wcf-logo-color.svg` at 120 px as the header image.
- Don't redraw, recolour or modify these files. The text is already outlined, so they render the same everywhere.
- The logo is app branding only. It never appears on the cards.

## Card specification (match `reference/index.html` exactly)
- CR80 portrait card, 54 × 85.6 mm. Design at 540 × 856 CSS px (10 px per mm).
- **Export:** 600 DPI = 1276 × 2022 px, so html-to-image `pixelRatio` = 1276/540.
  - No bleed.
  - Leave out the slot guide and the rounded corners in exports; they appear in the preview only.
  - Wait for the fonts and images to load before rendering.
- **PDF per employee:** 2 pages (front, then back), each page 54 × 85.6 mm, with the card image filling the page. RGB output; the printer converts it to CMYK.
- **Colour tokens:**
  - charcoal `#17202A`
  - back navy `#12203A`
  - off-white `#F7F6F2`
  - coral `#D98282`
  - gold `#E5B95C`
  - Dynamis navy `#0B2A55`
- **Front:**
  - charcoal background
  - client logo, top-centred, fitted inside 380 × 170 px
  - photo, 250 × 270 px, rounded 12 px, using the crop settings (`object-position` plus `scale`), or a silhouette when there is none
  - name, 42 px, weight 600
  - role, 24 px
  - EMPLOYEE ID and JOIN DATE labels in coral
  - an off-white footer with a coral diagonal stripe and the three tagline lines
- **Back** (fixed Dynamis content):
  - navy background
  - white "D" with the gold chevron, and the DYNAMIS wordmark with its custom "Λ" A
  - "PEOPLE | PLACEMENT | PROGRESS"
  - address, phone and email with icons
  - the two property/return notices
  - a gold diagonal accent

## Claude prompt shown in the Add client form (verbatim)
```
I'm preparing a client logo for an ID card printing system. The attached image is the client's logo, and it may be a sheet showing several versions.
Create ONE SVG file for the card front. The card background is dark charcoal (#17202A).
Requirements:
1. Choose the lockup that has both the symbol and the company name and fits a 38 × 17 mm area (aspect ratio 1:1 to 4:1, stacked or horizontal).
2. The colours must read clearly on #17202A. Keep the brand colours if they have enough contrast; otherwise use white (#FFFFFF).
3. Convert all text to paths. Use no <text> elements and no font references. Trace the letterforms closely.
4. Include a viewBox, crop tightly, and leave out width and height attributes.
5. Use a transparent background, with no background rectangle.
6. Use no <image>, embedded bitmaps, external links, scripts, filters, masks or <style> blocks. Use hex fill attributes only.
7. Keep the file lean: merge paths where possible and round coordinates to 1 decimal.
Output the complete SVG in one code block, named <client-name>-logo-dark.svg. Then render a preview on a #17202A background.
```

## Security and robustness
- Limit request bodies to 10 MB.
- Protect all paths against traversal, and validate `employee_id` against `^[A-Za-z0-9_-]{1,32}$`.
- Escape all user text before rendering it. Check the role on every API call on the server.
- Return JSON errors with clear messages that say what went wrong and how to fix it.
- Shut down gracefully on Ctrl+C, finishing any writes in progress.

## Tests
- Write `go test` tests for:
  - storage (atomic writes, concurrent writers)
  - PIN hashing and lockout
  - sessions and role checks
  - SVG validation
  - export filenames and paths
  - Excel date parsing, if done on the server
- Run `go vet` and `go test ./...` before every commit.
- Smoke test: start the binary against a temp data folder, create the admin, sign in, add a client, add an employee, and hit the export endpoint using `curl`.

## Task list (copy to PROGRESS.md as checkboxes)
1. Check prerequisites, run `git init` (if needed), extract Appendix A into `reference/index.html` and Appendix B into `web/assets/brand/`, and add `.gitignore` (`dist/`, `data/`, `config.json`), `go.mod` and the skeleton layout.
2. Config loading and the portable data folder, with atomic JSON storage and tests.
3. Auth: PIN hashing, first-run setup, login and logout, sessions, lockout, role middleware, and tests.
4. Users API (list, add, reset PIN, enable/disable, last-admin guard), with tests.
5. Activity log: write, and query with filters and pagination.
6. Presence tracking with reverse-DNS caching, and a server-info endpoint.
7. Clients API with SVG validation and sanitising, and the Helios seed. Tests.
8. Employees API (list, upsert, bulk import, map client) and the photo upload endpoint.
9. Export endpoint that saves PDFs into month folders and records exports. Tests.
10. `scripts/vendor.*`, fetching and committing the pinned libraries and fonts.
11. Frontend shell: embed, first-run setup, login, top bar, routing, API client and presence heartbeat.
12. Users page and Activity page.
13. Card renderer module, matching the reference exactly, with the live preview and navigation.
14. Wizard Step 1: clients dropdown, add/edit, SVG upload or paste, the Claude prompt, and live preview.
15. `scripts/make_template.py` and the committed Excel template.
16. Wizard Step 2: Excel/CSV import, the Needs attention mapping, the add-employee form, search and filter.
17. Wizard Step 3: photo queue, upload and resize, crop sliders and drag, save and next.
18. Wizard Step 4: filters, grouped selection with shift-range, and PDF/ZIP export with progress.
19. Under the hood panel.
20. `scripts/build.*`: cross-compile and test all four binaries.
21. End-to-end smoke test, README (setup, build, running on a LAN, backup by copying `data/`, the Windows OneDrive Desktop note, troubleshooting), cleanup pass, and the Handover section.

## Appendix A: reference/index.html (approved card templates, logos and tokens)

````html
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>ID Card Studio · White Coat Foundry</title>
<link href="https://fonts.googleapis.com/css2?family=Josefin+Sans:wght@400;600&family=Montserrat:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
:root{--charcoal:#17202A;--navy:#12203A;--offwhite:#F7F6F2;--coral:#D98282;--gold:#E5B95C;--dnavy:#0B2A55;--muted:#AEB4BC}
*{box-sizing:border-box;margin:0}
body{display:grid;grid-template-columns:340px 1fr;height:100vh;background:#3f4247;font-family:Montserrat,sans-serif}

/* App */
aside{background:#F4F4F2;color:var(--charcoal);overflow:auto;padding:24px;display:flex;flex-direction:column;gap:28px}
aside h2{font-size:15px;font-weight:700}
aside label{display:block;font-size:12px;color:#5a6068;margin-top:10px}
aside input:not([type=range]),aside select,aside textarea{width:100%;font:inherit;font-size:14px;padding:7px 9px;border:1px solid #cfd2d6;border-radius:6px;background:#fff}
aside input[type=range]{width:100%}
.btns{display:flex;flex-wrap:wrap;gap:8px;margin-top:12px}
.btn{display:inline-flex;align-items:center;font:600 13px/1.2 Montserrat,sans-serif;padding:8px 12px;border-radius:6px;border:1px solid var(--charcoal);background:var(--charcoal);color:#fff;cursor:pointer}
.btn.ghost{background:transparent;color:var(--charcoal)}
.btn:focus-visible{outline:3px solid var(--gold);outline-offset:2px}
.list{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}
.list button{font:inherit;font-size:12px;padding:4px 10px;border-radius:12px;border:1px solid #cfd2d6;background:#fff;cursor:pointer}
.list button.sel{border-color:var(--charcoal)}
#status{font-size:12px;line-height:1.5;color:#9b3d3d;white-space:pre-line}
main{overflow:auto;padding:32px}
#cards{display:flex;flex-wrap:wrap;gap:24px}
.empty{color:#cfd3d8;font-size:15px}
.pair{display:flex;gap:16px;zoom:.42;cursor:pointer}
.pair.sel .card{box-shadow:0 0 0 10px var(--gold)}
.hidden{display:none!important}

/* Card: 54 x 85.6 mm at 10 px/mm */
.card{width:540px;height:856px;position:relative;overflow:hidden;border-radius:32px}
.slot{position:absolute;top:30px;left:50%;width:120px;height:22px;margin-left:-60px;border-radius:11px;background:#6b7078}

/* Front */
.front{background:var(--charcoal);color:#fff;text-align:center}
.brand{position:absolute;top:70px;left:0;right:0;display:flex;justify-content:center}
.brand img{max-width:380px;max-height:170px}
.brand .name{color:var(--coral);font-size:28px;font-weight:600;letter-spacing:.1em;margin-top:60px}
.pair .photo{cursor:copy}
.photo{position:absolute;top:250px;left:145px;width:250px;height:270px;border-radius:12px;overflow:hidden;background:#d9dcdf}
.photo img,.photo svg{width:100%;height:100%;object-fit:cover}
.front h1{position:absolute;top:540px;left:30px;right:30px;font-size:42px;font-weight:600;letter-spacing:-.01em}
.role{position:absolute;top:594px;left:30px;right:30px;font-size:24px;color:#D5D9DE}
.meta{position:absolute;top:638px;left:0;right:0;display:flex;justify-content:center;gap:44px;font-size:17px}
.meta dt{color:var(--coral);font-size:13px;letter-spacing:.12em;margin-bottom:2px}
.meta dd{font-weight:600}
.foot,.stripe{position:absolute;left:0;right:0;bottom:0;height:170px}
.foot{background:var(--offwhite);clip-path:polygon(0 50px,76% 50px,100% 10px,100% 100%,0 100%)}
.stripe{background:var(--coral);clip-path:polygon(71% 50px,76% 50px,100% 10px,100% 0)}
.foot p{position:absolute;left:40px;top:78px;text-align:left;color:var(--charcoal);font-size:15px;font-weight:600;letter-spacing:.3em;line-height:1.55}

/* Helios logo */
.helios{display:flex;flex-direction:column;align-items:center;gap:12px;color:var(--coral);font-family:'Josefin Sans',sans-serif}
.helios svg{width:96px}
.helios b{font-weight:600;font-size:22px;letter-spacing:.32em;margin-right:-.32em}
.helios span{font-size:17px;letter-spacing:.42em;margin-right:-.42em;margin-top:-4px}

/* Back */
.back{background:var(--navy);color:#fff;padding:0 48px}
.back::before{content:"";position:absolute;top:-120px;right:-160px;width:420px;height:420px;background:#fff;opacity:.03;transform:rotate(45deg)}
.back .gold{position:absolute;right:-40px;bottom:120px;width:190px;height:34px;background:linear-gradient(90deg,#E9B23E,#B57A1C);transform:rotate(-52deg)}
.dynamis{display:flex;flex-direction:column;align-items:center;gap:12px;margin-top:80px;--d:#fff}
.dynamis .mark{width:112px}
.dynamis .word{font-weight:600;font-size:40px;letter-spacing:.14em;margin-right:-.14em;line-height:1}
.dynamis .word svg{height:.7em;margin-right:.14em;vertical-align:0}
.tag{text-align:center;font-size:14px;font-weight:600;letter-spacing:.24em;margin:18px -20px 0;color:#D5D9DE}
.contact{margin-top:44px;font-size:19px;line-height:1.45;display:grid;grid-template-columns:28px 1fr;gap:18px 16px;align-items:start}
.contact svg{width:22px;fill:#fff;margin-top:3px}
.contact .addr{margin-bottom:10px}
.back hr{border:0;border-top:1px solid #ffffff40;margin:28px 0 22px}
.legal{font-size:16px;line-height:1.45;color:#D5D9DE}
.legal+.legal{margin-top:14px}
</style>
<script src="https://cdnjs.cloudflare.com/ajax/libs/PapaParse/5.4.1/papaparse.min.js"></script>
</head>
<body>

<svg width="0" height="0" style="position:absolute" aria-hidden="true">
  <defs>
    <linearGradient id="hg" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#F2C66D"/><stop offset="1" stop-color="#E7757E"/></linearGradient>
    <linearGradient id="dg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#E9B23E"/><stop offset="1" stop-color="#B57A1C"/></linearGradient>
    <symbol id="helios-mark" viewBox="0 0 160 160">
      <path id="rays" fill="none" stroke="url(#hg)" stroke-width="2.6" stroke-linecap="round"/>
      <path d="M71 67v26M89 67v26M71 80h18" fill="none" stroke="url(#hg)" stroke-width="4" stroke-linecap="round"/>
    </symbol>
    <symbol id="dynamis-mark" viewBox="0 0 112 100">
      <path style="fill:var(--d)" d="M0 0H62A50 50 0 0 1 62 100H30L43 82H62A32 32 0 0 0 62 18H18Z"/>
      <path fill="url(#dg)" d="M14 28H36L60 52L26 100H4L38 52Z"/>
    </symbol>
    <symbol id="dynamis-a" viewBox="0 0 40 36">
      <path style="fill:var(--d)" d="M16 0H24L40 36H33L20 7L7 36H0Z"/>
      <path fill="url(#dg)" d="M20 22L26 36H14Z"/>
    </symbol>
    <symbol id="silhouette" viewBox="0 0 250 270">
      <rect width="250" height="270" fill="#d9dcdf"/>
      <circle cx="125" cy="105" r="52" fill="#b9bec4"/><path d="M30 270c0-62 42-100 95-100s95 38 95 100z" fill="#b9bec4"/>
    </symbol>
  </defs>
</svg>

<aside>
  <section>
    <h2>Employees</h2>
    <div class="btns">
      <label class="btn">Import CSV<input type="file" id="csv" accept=".csv" hidden></label>
      <label class="btn ghost">Add photos<input type="file" id="photos" accept="image/*" multiple hidden></label>
    </div>
    <p id="status"></p>
    <form id="emp" autocomplete="off">
      <label>Employee ID<input name="employee_id" required></label>
      <label>Name<input name="name" required></label>
      <label>Role<input name="role"></label>
      <label>Client<select name="client"></select></label>
      <label>Join date<input name="join_date" type="date"></label>
      <label>Photo<input type="file" id="photo1" accept="image/*"></label>
      <label>Photo zoom<input name="zoom" type="range" min="1" max="2.5" step=".05" value="1"></label>
      <label>Photo horizontal<input name="x" type="range" min="0" max="100" value="50"></label>
      <label>Photo vertical<input name="y" type="range" min="0" max="100" value="50"></label>
      <div class="btns">
        <button class="btn" id="add">Add employee</button>
        <button type="button" class="btn ghost hidden" id="done">Done</button>
        <button type="button" class="btn ghost hidden" id="del">Delete</button>
      </div>
    </form>
  </section>

  <section>
    <h2>Clients</h2>
    <div class="list" id="clist"></div>
    <form id="cli" autocomplete="off">
      <label>Client name<input name="name" required></label>
      <label>Logo<input type="file" id="logo" accept="image/*"></label>
      <label>Footer tagline (3 lines)<textarea name="tagline" rows="3"></textarea></label>
      <div class="btns">
        <button class="btn">Save client</button>
        <button type="button" class="btn ghost" id="cnew">New</button>
        <button type="button" class="btn ghost" id="cdel">Delete</button>
      </div>
    </form>
  </section>

  <section>
    <h2>Project</h2>
    <div class="btns">
      <button type="button" class="btn ghost" id="save">Save project</button>
      <label class="btn ghost">Open project<input type="file" id="open" accept=".json" hidden></label>
    </div>
  </section>
</aside>

<main><div id="cards"></div><input type="file" id="pick" accept="image/*" hidden></main>

<script>
const BACK={
  tagline:'PEOPLE  |  PLACEMENT  |  PROGRESS',
  address:'Krishna Complex, Plot No. 2,<br>Property No. 545/1,<br>Near Chambharli Naka, At Rees<br>(Navin Vasahat), Post Mohopada,<br>Taluka Khalapur, District Raigad,<br>Maharashtra – 410222',
  phone:'+91&nbsp;7483831212',
  email:'Dynamis@rautgroup.com',
  legal:['This card remains the property of Dynamis and must be returned upon cessation of employment.','If found, please return to the above address or contact +91&nbsp;7483831212.']
};

let CLIENTS={helios:{name:'Helios Material Handling',logo:'builtin',tagline:['ASSETS','PEOPLE','PERFORMANCE']}};
let ROWS=[],PHOTOS={},sel=null,csel=null;

const esc=s=>String(s??'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));
const fmtDate=s=>/^\d{4}-\d{2}-\d{2}$/.test(s)?new Date(s+'T00:00').toLocaleDateString('en-GB',{day:'2-digit',month:'short',year:'numeric'}):esc(s);
const use=(id,vb)=>`<svg viewBox="${vb}"><use href="#${id}"/></svg>`;

const HELIOS=`<div class="helios">${use('helios-mark','0 0 160 160')}<b>HELIOS MATERIAL</b><span>HANDLING</span></div>`;
const DYNAMIS=`<div class="dynamis">${use('dynamis-mark','0 0 112 100').replace('<svg','<svg class="mark"')}<div class="word">DYN${use('dynamis-a','0 0 40 36')}MIS</div></div>`;
const ICON={
  pin:'<svg viewBox="0 0 24 24"><path d="M12 2a7 7 0 0 0-7 7c0 5 7 13 7 13s7-8 7-13a7 7 0 0 0-7-7zm0 9.5a2.5 2.5 0 1 1 0-5 2.5 2.5 0 0 1 0 5z"/></svg>',
  phone:'<svg viewBox="0 0 24 24"><path d="M6.6 10.8a15 15 0 0 0 6.6 6.6l2.2-2.2a1 1 0 0 1 1-.25 11 11 0 0 0 3.6.6 1 1 0 0 1 1 1V20a1 1 0 0 1-1 1A17 17 0 0 1 3 4a1 1 0 0 1 1-1h3.5a1 1 0 0 1 1 1 11 11 0 0 0 .6 3.6 1 1 0 0 1-.25 1z"/></svg>',
  mail:'<svg viewBox="0 0 24 24"><path d="M2 5h20v14H2zm2 2v.4l8 5.6 8-5.6V7zm0 2.8V17h16V9.8l-8 5.6z" fill-rule="evenodd"/></svg>'
};

function front(r){
  const c=CLIENTS[r.client]||{name:r.client,tagline:[]};
  const logo=c.logo==='builtin'?HELIOS:c.logo?`<img src="${c.logo}" alt="${esc(c.name)}">`:`<p class="name">${esc(c.name)}</p>`;
  const src=photoOf(r),z=r.zoom||1,x=r.x??50,y=r.y??50;
  const photo=src?`<img src="${src}" alt="" style="object-position:${x}% ${y}%;transform:scale(${z});transform-origin:${x}% ${y}%">`:use('silhouette','0 0 250 270');
  return `<div class="card front"><div class="slot"></div>
    <div class="brand">${logo}</div>
    <div class="photo">${photo}</div>
    <h1>${esc(r.name)}</h1><p class="role">${esc(r.role)}</p>
    <dl class="meta"><div><dt>EMPLOYEE ID</dt><dd>${esc(r.employee_id)}</dd></div><div><dt>JOIN DATE</dt><dd>${fmtDate(r.join_date)}</dd></div></dl>
    <div class="stripe"></div>
    <div class="foot"><p>${c.tagline.map(esc).join('<br>')}</p></div>
  </div>`;
}

const BACK_HTML=`<div class="card back"><div class="slot"></div><div class="gold"></div>
  ${DYNAMIS}<p class="tag">${BACK.tagline}</p>
  <div class="contact">${ICON.pin}<p class="addr">${BACK.address}</p>${ICON.phone}<p>${BACK.phone}</p>${ICON.mail}<p>${BACK.email}</p></div>
  <hr>${BACK.legal.map(t=>`<p class="legal">${t}</p>`).join('')}
</div>`;

// Sunburst rays
{const d=[],L=[16,9,13,6,11];
for(let i=0;i<56;i++){const a=i*Math.PI/28,c=Math.cos(a),s=Math.sin(a),p=r=>`${(80+r*c).toFixed(1)} ${(80+r*s).toFixed(1)}`;
  d.push(`M${p(42)}L${p(42+L[i%5]*.8)}M${p(60)}L${p(i%2?70:74)}`);}
document.getElementById('rays').setAttribute('d',d.join(''));}


const $=id=>document.getElementById(id),emp=$('emp'),cli=$('cli');
const key=s=>String(s??'').trim().toLowerCase();
const photoOf=r=>r.img||PHOTOS[key(r.photo)]||Object.entries(PHOTOS).find(([k])=>k.split('.')[0]===key(r.employee_id))?.[1];
const readURL=f=>new Promise(res=>{const fr=new FileReader();fr.onload=()=>res(fr.result);fr.readAsDataURL(f)});
const findClient=v=>Object.keys(CLIENTS).find(k=>k===key(v)||key(CLIENTS[k].name)===key(v));

function render(){
  $('cards').innerHTML=ROWS.length?ROWS.map((r,i)=>`<div class="pair${i===sel?' sel':''}" data-i="${i}">${front(r)}${BACK_HTML}</div>`).join('')
    :'<p class="empty">Import a CSV or add an employee to see cards.</p>';
  const missing=ROWS.filter(r=>!CLIENTS[r.client]).map(r=>`${r.employee_id}: unknown client "${r.client}"`);
  $('status').textContent=missing.join('\n');
}
function renderClients(){
  emp.client.innerHTML=Object.entries(CLIENTS).map(([k,c])=>`<option value="${k}">${esc(c.name)}</option>`).join('');
  $('clist').innerHTML=Object.entries(CLIENTS).map(([k,c])=>`<button type="button" data-k="${k}" class="${k===csel?'sel':''}">${esc(c.name)}</button>`).join('');
}

// Employees
const FIELDS=['employee_id','name','role','client','join_date','zoom','x','y'];
function formRow(){const r={};FIELDS.forEach(f=>r[f]=emp[f].value.trim());['zoom','x','y'].forEach(f=>r[f]=+r[f]);return r}
function select(i){
  sel=i;const r=ROWS[i]||{};
  emp.reset();FIELDS.forEach(f=>{if(r[f]!=null)emp[f].value=r[f]});
  ['add','done','del'].forEach(b=>$(b).classList.toggle('hidden',(b==='add')===(i!=null)));
  render();
}
emp.oninput=e=>{if(sel==null||e.target.type==='file')return;Object.assign(ROWS[sel],formRow());render()};
emp.onsubmit=e=>{e.preventDefault();ROWS.push({...formRow(),img:emp.dataset.img||''});delete emp.dataset.img;select(null)};
$('done').onclick=()=>select(null);
$('del').onclick=()=>{ROWS.splice(sel,1);select(null)};
// Per-employee photo: form field, click on card photo, or drop onto card
let target=null;
const setImg=async(i,f)=>{if(f?.type.startsWith('image/')){ROWS[i].img=await readURL(f);render()}};
$('cards').onclick=e=>{
  const p=e.target.closest('.pair');if(!p)return;
  if(e.target.closest('.photo')){target=+p.dataset.i;$('pick').click()}else select(+p.dataset.i);
};
$('pick').onchange=e=>{setImg(target,e.target.files[0]);e.target.value=''};
$('cards').ondragover=e=>e.preventDefault();
$('cards').ondrop=e=>{e.preventDefault();const p=e.target.closest('.pair');if(p)setImg(+p.dataset.i,e.dataTransfer.files[0])};
$('photo1').onchange=async e=>{
  const f=e.target.files[0];if(!f)return;
  if(sel!=null)setImg(sel,f);else emp.dataset.img=await readURL(f);
  e.target.value='';
};
$('photos').onchange=async e=>{for(const f of e.target.files)PHOTOS[key(f.name)]=await readURL(f);e.target.value='';render()};
$('csv').onchange=e=>{
  Papa.parse(e.target.files[0],{header:true,skipEmptyLines:true,transformHeader:key,complete:({data})=>{
    ROWS.push(...data.map(d=>({...d,client:findClient(d.client)||d.client,zoom:1,x:50,y:50})));
    e.target.value='';render();
  }});
};

// Clients
function cselect(k){
  csel=k;const c=CLIENTS[k];cli.reset();
  if(c){cli.name.value=c.name;cli.tagline.value=c.tagline.join('\n')}
  renderClients();
}
$('clist').onclick=e=>{const b=e.target.closest('button');if(b)cselect(b.dataset.k)};
$('cnew').onclick=()=>cselect(null);
$('cdel').onclick=()=>{if(csel){delete CLIENTS[csel];cselect(null);render()}};
cli.onsubmit=async e=>{
  e.preventDefault();
  const k=csel||key(cli.name.value).replace(/[^a-z0-9]+/g,'-'),f=$('logo').files[0];
  CLIENTS[k]={...CLIENTS[k],name:cli.name.value.trim(),tagline:cli.tagline.value.split('\n').map(s=>s.trim().toUpperCase()).filter(Boolean).slice(0,3)};
  if(f)CLIENTS[k].logo=await readURL(f);
  cselect(k);render();
};

// Project
$('save').onclick=()=>{
  const a=document.createElement('a');
  a.href=URL.createObjectURL(new Blob([JSON.stringify({CLIENTS,ROWS,PHOTOS})],{type:'application/json'}));
  a.download='id-card-project.json';a.click();URL.revokeObjectURL(a.href);
};
$('open').onchange=async e=>{
  ({CLIENTS,ROWS,PHOTOS}=JSON.parse(await e.target.files[0].text()));
  e.target.value='';cselect(null);select(null);
};

renderClients();render();
</script>
</body>
</html>
````

## Appendix B: White Coat Foundry logos (official, use as-is)

`web/assets/brand/wcf-logo-color.svg`

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000" role="img"><title>White Coat Foundry</title><circle cx="500" cy="500" r="470" fill="none" stroke="#191919" stroke-width="9"/><circle cx="500" cy="500" r="292" fill="none" stroke="#191919" stroke-width="9"/><g fill="#191919"><path transform="translate(198.33 332.61) rotate(-60.98)" d="M-26.728 0.0 -50.336 -73.42399999999999H-34.111999999999995L-16.848 -16.951999999999998H-23.192L-6.344000000000001 -73.42399999999999H6.448L23.295999999999992 -16.951999999999998H17.055999999999997L34.215999999999994 -73.42399999999999H50.336L26.727999999999994 0.0H13.935999999999993L-3.0159999999999982 -56.471999999999994H3.0159999999999982L-13.936 0.0Z"/><path transform="translate(266.05 246.44) rotate(-42.7)" d="M-30.888 0.0V-73.42399999999999H-14.560000000000002V0.0ZM14.559999999999995 0.0V-73.42399999999999H30.991999999999997V0.0ZM-21.112000000000002 -30.68V-44.824H19.967999999999996V-30.68Z"/><path transform="translate(324.69 202.86) rotate(-30.54)" d="M-8.163999999999998 0.0V-73.42399999999999H8.164V0.0Z"/><path transform="translate(386.71 174.13) rotate(-19.17)" d="M-8.164000000000001 0.0V-72.384H8.163999999999994V0.0ZM-31.148000000000003 -59.175999999999995V-73.42399999999999H31.14799999999999V-59.175999999999995Z"/><path transform="translate(469.81 156.32) rotate(-5.02)" d="M-24.907999999999998 0.0V-73.42399999999999H-8.579999999999998V0.0ZM-12.427999999999997 0.0V-14.144H27.82V0.0ZM-12.427999999999997 -30.68V-44.303999999999995H24.18V-30.68ZM-12.427999999999997 -59.384V-73.42399999999999H27.299999999999997V-59.384Z"/><path transform="translate(596.05 168.64) rotate(16.17)" d="M6.084000000000003 1.144Q-2.131999999999998 1.144 -9.151999999999997 -1.7159999999999997Q-16.171999999999997 -4.576 -21.423999999999996 -9.776Q-26.675999999999995 -14.975999999999999 -29.587999999999994 -21.892Q-32.49999999999999 -28.808 -32.49999999999999 -36.815999999999995Q-32.49999999999999 -44.824 -29.587999999999994 -51.739999999999995Q-26.675999999999995 -58.656 -21.475999999999996 -63.751999999999995Q-16.275999999999996 -68.848 -9.255999999999997 -71.708Q-2.235999999999997 -74.568 5.980000000000004 -74.568Q14.716000000000001 -74.568 21.528 -71.75999999999999Q28.339999999999996 -68.952 33.436 -64.064L22.516 -53.04Q19.604 -56.16 15.496000000000002 -57.928Q11.388000000000005 -59.696 5.980000000000004 -59.696Q1.196000000000005 -59.696 -2.8079999999999963 -58.083999999999996Q-6.811999999999998 -56.471999999999994 -9.671999999999997 -53.455999999999996Q-12.531999999999996 -50.44 -14.143999999999997 -46.176Q-15.755999999999997 -41.912 -15.755999999999997 -36.815999999999995Q-15.755999999999997 -31.616 -14.143999999999997 -27.351999999999997Q-12.531999999999996 -23.087999999999997 -9.671999999999997 -20.072Q-6.811999999999998 -17.056 -2.8079999999999963 -15.392Q1.196000000000005 -13.728 5.980000000000004 -13.728Q11.596000000000004 -13.728 15.756000000000004 -15.495999999999999Q19.916000000000004 -17.264 22.932000000000002 -20.384L33.955999999999996 -9.36Q28.652000000000008 -4.4719999999999995 21.840000000000003 -1.6639999999999997Q15.027999999999999 1.144 6.084000000000003 1.144Z"/><path transform="translate(684.85 208.7) rotate(32.4)" d="M0.2079999999999984 1.248Q-8.112000000000002 1.248 -15.132000000000001 -1.6639999999999997Q-22.152 -4.576 -27.456 -9.776Q-32.76 -14.975999999999999 -35.672 -21.892Q-38.583999999999996 -28.808 -38.583999999999996 -36.815999999999995Q-38.583999999999996 -44.928 -35.672 -51.792Q-32.76 -58.656 -27.56 -63.804Q-22.36 -68.952 -15.34 -71.812Q-8.32 -74.672 0.0 -74.672Q8.216000000000001 -74.672 15.235999999999997 -71.812Q22.255999999999993 -68.952 27.507999999999996 -63.804Q32.76 -58.656 35.672 -51.739999999999995Q38.583999999999996 -44.824 38.583999999999996 -36.711999999999996Q38.583999999999996 -28.703999999999997 35.672 -21.787999999999997Q32.76 -14.872 27.559999999999995 -9.724Q22.359999999999992 -4.576 15.339999999999996 -1.6639999999999997Q8.32 1.248 0.2079999999999984 1.248ZM0.0 -13.623999999999999Q6.552 -13.623999999999999 11.491999999999997 -16.536Q16.431999999999995 -19.448 19.135999999999996 -24.7Q21.839999999999996 -29.951999999999998 21.839999999999996 -36.815999999999995Q21.839999999999996 -42.016 20.279999999999998 -46.227999999999994Q18.72 -50.44 15.808 -53.507999999999996Q12.896 -56.576 8.892 -58.188Q4.887999999999998 -59.8 0.0 -59.8Q-6.552 -59.8 -11.491999999999999 -56.94Q-16.432 -54.08 -19.136 -48.932Q-21.84 -43.784 -21.84 -36.815999999999995Q-21.84 -31.616 -20.28 -27.351999999999997Q-18.72 -23.087999999999997 -15.86 -20.019999999999996Q-13.0 -16.951999999999998 -8.943999999999999 -15.287999999999998Q-4.887999999999998 -13.623999999999999 0.0 -13.623999999999999Z"/><path transform="translate(760.28 273.55) rotate(48.98)" d="M-36.348 0.0 -7.2280000000000015 -73.42399999999999H7.539999999999999L36.348 0.0H19.083999999999996L-2.9640000000000057 -60.943999999999996H2.9639999999999986L-19.396 0.0ZM-19.812 -13.312V-26.624H20.227999999999994V-13.312Z"/><path transform="translate(810.38 349.36) rotate(64.11)" d="M-8.164000000000001 0.0V-72.384H8.163999999999994V0.0ZM-31.148000000000003 -59.175999999999995V-73.42399999999999H31.14799999999999V-59.175999999999995Z"/><path transform="translate(236.19 822.94) rotate(39.25)" d="M-23.608 0.0V-73.42399999999999H-7.280000000000001V0.0ZM-11.128 -28.808V-42.848H25.583999999999996V-28.808ZM-11.128 -59.384V-73.42399999999999H27.247999999999998V-59.384Z"/><path transform="translate(313.78 873.11) rotate(26.52)" d="M0.2079999999999984 1.248Q-8.112000000000002 1.248 -15.132000000000001 -1.6639999999999997Q-22.152 -4.576 -27.456 -9.776Q-32.76 -14.975999999999999 -35.672 -21.892Q-38.583999999999996 -28.808 -38.583999999999996 -36.815999999999995Q-38.583999999999996 -44.928 -35.672 -51.792Q-32.76 -58.656 -27.56 -63.804Q-22.36 -68.952 -15.34 -71.812Q-8.32 -74.672 0.0 -74.672Q8.216000000000001 -74.672 15.235999999999997 -71.812Q22.255999999999993 -68.952 27.507999999999996 -63.804Q32.76 -58.656 35.672 -51.739999999999995Q38.583999999999996 -44.824 38.583999999999996 -36.711999999999996Q38.583999999999996 -28.703999999999997 35.672 -21.787999999999997Q32.76 -14.872 27.559999999999995 -9.724Q22.359999999999992 -4.576 15.339999999999996 -1.6639999999999997Q8.32 1.248 0.2079999999999984 1.248ZM0.0 -13.623999999999999Q6.552 -13.623999999999999 11.491999999999997 -16.536Q16.431999999999995 -19.448 19.135999999999996 -24.7Q21.839999999999996 -29.951999999999998 21.839999999999996 -36.815999999999995Q21.839999999999996 -42.016 20.279999999999998 -46.227999999999994Q18.72 -50.44 15.808 -53.507999999999996Q12.896 -56.576 8.892 -58.188Q4.887999999999998 -59.8 0.0 -59.8Q-6.552 -59.8 -11.491999999999999 -56.94Q-16.432 -54.08 -19.136 -48.932Q-21.84 -43.784 -21.84 -36.815999999999995Q-21.84 -31.616 -20.28 -27.351999999999997Q-18.72 -23.087999999999997 -15.86 -20.019999999999996Q-13.0 -16.951999999999998 -8.943999999999999 -15.287999999999998Q-4.887999999999998 -13.623999999999999 0.0 -13.623999999999999Z"/><path transform="translate(406.19 906.31) rotate(13)" d="M0.1039999999999992 1.144Q-8.943999999999999 1.144 -15.859999999999998 -2.7039999999999997Q-22.775999999999996 -6.552 -26.675999999999995 -13.363999999999999Q-30.575999999999997 -20.176 -30.575999999999997 -28.912V-73.42399999999999H-14.143999999999998V-27.976Q-14.143999999999998 -23.608 -12.271999999999998 -20.384Q-10.399999999999999 -17.16 -7.123999999999999 -15.443999999999999Q-3.847999999999999 -13.728 0.1039999999999992 -13.728Q4.264000000000003 -13.728 7.384 -15.443999999999999Q10.503999999999998 -17.16 12.323999999999998 -20.332Q14.143999999999998 -23.503999999999998 14.143999999999998 -27.872V-73.42399999999999H30.576V-28.808Q30.576 -20.072 26.728 -13.312Q22.880000000000003 -6.552 16.016000000000002 -2.7039999999999997Q9.152000000000001 1.144 0.1039999999999992 1.144Z"/><path transform="translate(500.26 917) rotate(-0.04)" d="M-31.304 0.0V-73.42399999999999H-19.863999999999997L-14.975999999999999 -58.76V0.0ZM19.344 0.0 -23.919999999999998 -55.431999999999995 -19.863999999999997 -73.42399999999999 23.4 -17.992ZM19.344 0.0 14.975999999999999 -14.664V-73.42399999999999H31.303999999999995V0.0Z"/><path transform="translate(596.95 905.57) rotate(-13.44)" d="M-21.112 0.0V-14.351999999999999H-2.911999999999999Q3.7439999999999998 -14.351999999999999 8.736 -17.003999999999998Q13.728000000000002 -19.656 16.432 -24.752Q19.135999999999996 -29.848 19.135999999999996 -36.815999999999995Q19.135999999999996 -43.784 16.38 -48.775999999999996Q13.624000000000002 -53.768 8.684000000000001 -56.471999999999994Q3.7439999999999998 -59.175999999999995 -2.911999999999999 -59.175999999999995H-21.631999999999998V-73.42399999999999H-2.7040000000000006Q5.616 -73.42399999999999 12.636 -70.77199999999999Q19.656 -68.11999999999999 24.907999999999998 -63.17999999999999Q30.159999999999997 -58.239999999999995 33.019999999999996 -51.532Q35.879999999999995 -44.824 35.879999999999995 -36.711999999999996Q35.879999999999995 -28.703999999999997 33.019999999999996 -21.944Q30.159999999999997 -15.184 24.959999999999997 -10.296Q19.759999999999998 -5.4079999999999995 12.739999999999998 -2.7039999999999997Q5.719999999999999 0.0 -2.496000000000002 0.0ZM-32.135999999999996 0.0V-73.42399999999999H-15.808V0.0Z"/><path transform="translate(684.35 874.04) rotate(-26.24)" d="M-14.351999999999997 -30.264V-42.327999999999996H1.1440000000000055Q6.032000000000004 -42.327999999999996 8.684000000000001 -44.824Q11.335999999999999 -47.32 11.335999999999999 -51.583999999999996Q11.335999999999999 -55.535999999999994 8.736 -58.135999999999996Q6.136000000000003 -60.736 1.2480000000000047 -60.736H-14.351999999999997V-73.42399999999999H3.1200000000000045Q10.399999999999999 -73.42399999999999 15.911999999999999 -70.66799999999999Q21.424 -67.91199999999999 24.544 -63.023999999999994Q27.664 -58.135999999999996 27.664 -51.791999999999994Q27.664 -45.344 24.544 -40.507999999999996Q21.424 -35.672 15.808 -32.967999999999996Q10.192 -30.264 2.6000000000000014 -30.264ZM-26.831999999999997 0.0V-73.42399999999999H-10.503999999999998V0.0ZM13.104 0.0 -9.775999999999996 -31.616 5.200000000000003 -35.672 32.44800000000001 0.0Z"/><path transform="translate(759.89 826.11) rotate(-38.55)" d="M-6.084 -27.352 -34.372 -73.42399999999999H-15.443999999999999L6.292000000000002 -36.192H-5.875999999999998L15.86 -73.42399999999999H34.37200000000001L5.876000000000005 -27.352ZM-8.059999999999999 0.0V-34.839999999999996H8.268V0.0Z"/></g><circle cx="119" cy="500" r="13" fill="#F7941D"/><circle cx="881" cy="500" r="13" fill="#F7941D"/><g transform="translate(283.5 249.86) scale(0.42) rotate(16.0 509.5 607.5)"><mask id="a" maskUnits="userSpaceOnUse" x="0" y="0" width="1060" height="1220"><rect width="1060" height="1220" fill="#fff"/><path d="M135 485C138.67 491.5 151.5 511 157 524C162.5 537 163.17 550 168 563C172.83 576 178.5 589 186 602C193.5 615 203 628 213 641C223 654 233.83 667.67 246 680C258.17 692.33 279.33 709.17 286 715" fill="none" stroke="#000" stroke-width="35.58" stroke-linecap="round" stroke-linejoin="round"/><circle cx="327.88" cy="750.61" r="36.86" fill="none" stroke="#000" stroke-width="30"/><path d="M845 316C846.5 323.5 850.83 346 854 361C857.17 376 861.5 390.83 864 406C866.5 421.17 868.83 436.83 869 452C869.17 467.17 866.83 482 865 497C863.17 512 861.5 526.83 858 542C854.5 557.17 846.33 580.33 844 588" fill="none" stroke="#000" stroke-width="32.39" stroke-linecap="round" stroke-linejoin="round"/><circle cx="815.77" cy="633.22" r="35.23" fill="none" stroke="#000" stroke-width="32"/></mask><path d="M263 442.9C263 442.9 307.9 394.9 335 376.5C362.2 358.1 393.8 343.5 425.7 332.6C457.6 321.7 492.7 314.6 526.7 311.2C560.6 307.8 596.3 308.2 629.5 312.3C662.8 316.4 726 335.9 726 335.9C726 336.1 752.9 270.4 748.9 246C745 221.6 723.5 203.1 702.3 189.7C681.2 176.3 650.7 169.4 622.1 165.8C593.4 162.2 561.4 164.8 530.4 168.1C499.4 171.3 466.7 178 436 185.2C405.4 192.4 374.4 200.2 346.4 211.1C318.3 222 290.8 234.3 267.8 250.6C244.8 266.9 220.6 287.7 208.4 308.9C196.2 330.1 185.4 355.7 194.5 378.1C203.6 400.5 263 443.1 263 443.1ZM385 1053C385 1053 432.5 1028.4 449.4 1009.8C466.3 991.2 478 966.4 486.3 941.4C494.5 916.5 498.2 887.3 499 860C499.8 832.7 496.7 803.6 491.2 777.8C485.7 751.9 477 727.4 466.1 704.7C455.1 682 441.4 661.2 425.7 641.5C410.1 621.7 391.9 603.6 372.2 586.2C352.4 568.7 330.3 552.7 307.2 536.9C284.1 521.2 257.1 507.1 233.7 491.6C210.3 476.1 184 461.8 166.8 443.9C149.5 426.1 135.6 406.2 130 384.7C124.5 363.2 127.5 337.3 133.7 315.1C140 292.9 152.8 270 167.4 251.4C182.1 232.9 201.1 217.4 221.7 203.8C242.3 190.1 266.3 179.3 291.1 169.5C316 159.7 343.4 152.1 370.8 144.8C398.2 137.5 427.3 131.4 455.7 125.7C484.1 120.1 513.3 114.2 541.1 110.9C569 107.7 596.9 104.9 622.9 106.2C648.9 107.5 674.3 110.5 697 118.7C719.7 126.8 741.9 139.1 759.2 155.1C776.5 171.1 792.9 192.9 800.9 214.5C808.9 236.1 811.1 261 807.2 284.8C803.2 308.5 789.2 332.9 777.2 357.2C765.3 381.5 748 406 735.5 430.5C722.9 455.1 711 479.7 702.1 504.5C693.1 529.2 686 554.1 681.9 579.2C677.9 604.2 676.1 629.4 678 654.9C679.8 680.3 684.7 706.4 693 731.8C701.3 757.1 713.4 783.7 727.8 807C742.2 830.4 759.9 853.6 779.3 872C798.7 890.3 820.9 907 844.2 917.4C867.5 927.7 919 934 919 934C919.1 934 925.2 881.9 927.9 855.3C930.7 828.7 933.5 801.7 935.7 774.6C937.9 747.4 940 720 941.2 692.5C942.4 665.1 943.3 637.5 943.1 610C942.9 582.5 942.1 554.9 940.1 527.7C938.1 500.4 935.3 473.2 931.1 446.4C926.8 419.6 921.5 392.9 914.6 366.9C907.7 340.8 899.5 314.7 889.6 290.3C879.7 265.8 868.4 241.7 855.2 220C842.1 198.3 827.3 177.6 810.7 160.1C794 142.6 775.7 126.9 755.4 114.8C735.1 102.7 712.5 93.9 688.8 87.7C665.1 81.5 639.3 78.7 613.1 77.4C586.9 76.1 559.2 77.6 531.6 80C504 82.4 475.4 86.7 447.6 91.7C419.7 96.7 391.4 102.6 364.5 110C337.6 117.4 310.8 125.8 286 136.1C261.3 146.4 237.3 158 215.9 171.7C194.6 185.5 174.7 201 157.9 218.7C141.2 236.3 126.6 256.3 115.3 277.5C104 298.8 95.6 322.2 90.3 346.3C85 370.3 83.4 396.4 83.6 421.9C83.8 447.5 87.1 473.9 91.7 499.4C96.2 525 103 550.3 110.9 575.2C118.8 600.1 128.5 624.7 139.1 648.9C149.7 673.2 161.7 697.1 174.3 720.7C186.8 744.4 200.6 767.7 214.4 790.7C228.3 813.7 243 836.4 257.6 858.8C272.1 881.2 287.2 903.3 301.7 925.2C316.2 947 330.9 968.6 344.8 989.9C358.6 1011.2 384.8 1053 384.8 1053Z" fill="#191919" fill-rule="evenodd" mask="url(#a)"/><path d="M590.6 310.1C561.6 309.2 533.4 310.7 505.1 315.1C476.8 319.5 450.1 325.8 420.6 336.3C391.1 346.8 351.4 362.9 328.2 378C304.9 393.2 282 411.6 280.9 427.5C279.7 443.3 301.1 457.6 321.2 473C341.3 488.3 378.3 502.5 401.5 519.7C424.7 536.9 444.4 555.3 460.6 576.2C476.7 597 485.9 619.9 498.5 644.7C511 669.6 523.3 697.2 535.6 725C548 752.9 562.3 783.2 572.4 811.6C582.4 840 592.4 869.5 595.8 895.7C599.2 921.8 600.2 947.1 592.7 968.5C585.3 989.8 570.3 1006.3 551.1 1023.7C531.8 1041.1 490.3 1057.6 477.2 1072.6C464.1 1087.7 460.3 1103.2 472.5 1113.8C484.7 1124.4 522.5 1133.2 550.5 1136.3C578.5 1139.4 611.4 1137 640.5 1132.4C669.5 1127.8 696.4 1118.4 724.8 1108.7C753.1 1099 785.7 1089.4 810.8 1074.2C835.8 1058.9 869.4 1031.7 875 1017C880.7 1002.4 864.8 993.1 844.5 986.1C824.3 979.2 778.5 983.7 753.6 975.4C728.7 967.1 709 954.2 695 936.1C681.1 918 678.1 892.1 669.9 866.8C661.6 841.6 653.7 812.9 645.6 784.6C637.6 756.3 627.4 726.4 621.5 696.9C615.5 667.4 609.5 636.9 609.8 607.4C610.1 577.8 614.3 547.8 623.1 519.6C631.8 491.3 650.3 462.9 662.3 437.7C674.2 412.4 691.8 387.6 694.6 368C697.4 348.5 696.5 330.1 679.1 320.5C661.8 310.8 619.6 311 590.6 310.1Z" fill="#F7941D" fill-rule="evenodd"/></g></svg>
```

`web/assets/brand/wcf-logo-subtle-dark.svg`

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000" role="img"><title>White Coat Foundry</title><circle cx="500" cy="500" r="470" fill="none" stroke="#9b9b96" stroke-width="15"/><circle cx="500" cy="500" r="292" fill="none" stroke="#9b9b96" stroke-width="15"/><g fill="#9b9b96"><path transform="translate(198.33 332.61) rotate(-60.98)" d="M-26.728 0.0 -50.336 -73.42399999999999H-34.111999999999995L-16.848 -16.951999999999998H-23.192L-6.344000000000001 -73.42399999999999H6.448L23.295999999999992 -16.951999999999998H17.055999999999997L34.215999999999994 -73.42399999999999H50.336L26.727999999999994 0.0H13.935999999999993L-3.0159999999999982 -56.471999999999994H3.0159999999999982L-13.936 0.0Z"/><path transform="translate(266.05 246.44) rotate(-42.7)" d="M-30.888 0.0V-73.42399999999999H-14.560000000000002V0.0ZM14.559999999999995 0.0V-73.42399999999999H30.991999999999997V0.0ZM-21.112000000000002 -30.68V-44.824H19.967999999999996V-30.68Z"/><path transform="translate(324.69 202.86) rotate(-30.54)" d="M-8.163999999999998 0.0V-73.42399999999999H8.164V0.0Z"/><path transform="translate(386.71 174.13) rotate(-19.17)" d="M-8.164000000000001 0.0V-72.384H8.163999999999994V0.0ZM-31.148000000000003 -59.175999999999995V-73.42399999999999H31.14799999999999V-59.175999999999995Z"/><path transform="translate(469.81 156.32) rotate(-5.02)" d="M-24.907999999999998 0.0V-73.42399999999999H-8.579999999999998V0.0ZM-12.427999999999997 0.0V-14.144H27.82V0.0ZM-12.427999999999997 -30.68V-44.303999999999995H24.18V-30.68ZM-12.427999999999997 -59.384V-73.42399999999999H27.299999999999997V-59.384Z"/><path transform="translate(596.05 168.64) rotate(16.17)" d="M6.084000000000003 1.144Q-2.131999999999998 1.144 -9.151999999999997 -1.7159999999999997Q-16.171999999999997 -4.576 -21.423999999999996 -9.776Q-26.675999999999995 -14.975999999999999 -29.587999999999994 -21.892Q-32.49999999999999 -28.808 -32.49999999999999 -36.815999999999995Q-32.49999999999999 -44.824 -29.587999999999994 -51.739999999999995Q-26.675999999999995 -58.656 -21.475999999999996 -63.751999999999995Q-16.275999999999996 -68.848 -9.255999999999997 -71.708Q-2.235999999999997 -74.568 5.980000000000004 -74.568Q14.716000000000001 -74.568 21.528 -71.75999999999999Q28.339999999999996 -68.952 33.436 -64.064L22.516 -53.04Q19.604 -56.16 15.496000000000002 -57.928Q11.388000000000005 -59.696 5.980000000000004 -59.696Q1.196000000000005 -59.696 -2.8079999999999963 -58.083999999999996Q-6.811999999999998 -56.471999999999994 -9.671999999999997 -53.455999999999996Q-12.531999999999996 -50.44 -14.143999999999997 -46.176Q-15.755999999999997 -41.912 -15.755999999999997 -36.815999999999995Q-15.755999999999997 -31.616 -14.143999999999997 -27.351999999999997Q-12.531999999999996 -23.087999999999997 -9.671999999999997 -20.072Q-6.811999999999998 -17.056 -2.8079999999999963 -15.392Q1.196000000000005 -13.728 5.980000000000004 -13.728Q11.596000000000004 -13.728 15.756000000000004 -15.495999999999999Q19.916000000000004 -17.264 22.932000000000002 -20.384L33.955999999999996 -9.36Q28.652000000000008 -4.4719999999999995 21.840000000000003 -1.6639999999999997Q15.027999999999999 1.144 6.084000000000003 1.144Z"/><path transform="translate(684.85 208.7) rotate(32.4)" d="M0.2079999999999984 1.248Q-8.112000000000002 1.248 -15.132000000000001 -1.6639999999999997Q-22.152 -4.576 -27.456 -9.776Q-32.76 -14.975999999999999 -35.672 -21.892Q-38.583999999999996 -28.808 -38.583999999999996 -36.815999999999995Q-38.583999999999996 -44.928 -35.672 -51.792Q-32.76 -58.656 -27.56 -63.804Q-22.36 -68.952 -15.34 -71.812Q-8.32 -74.672 0.0 -74.672Q8.216000000000001 -74.672 15.235999999999997 -71.812Q22.255999999999993 -68.952 27.507999999999996 -63.804Q32.76 -58.656 35.672 -51.739999999999995Q38.583999999999996 -44.824 38.583999999999996 -36.711999999999996Q38.583999999999996 -28.703999999999997 35.672 -21.787999999999997Q32.76 -14.872 27.559999999999995 -9.724Q22.359999999999992 -4.576 15.339999999999996 -1.6639999999999997Q8.32 1.248 0.2079999999999984 1.248ZM0.0 -13.623999999999999Q6.552 -13.623999999999999 11.491999999999997 -16.536Q16.431999999999995 -19.448 19.135999999999996 -24.7Q21.839999999999996 -29.951999999999998 21.839999999999996 -36.815999999999995Q21.839999999999996 -42.016 20.279999999999998 -46.227999999999994Q18.72 -50.44 15.808 -53.507999999999996Q12.896 -56.576 8.892 -58.188Q4.887999999999998 -59.8 0.0 -59.8Q-6.552 -59.8 -11.491999999999999 -56.94Q-16.432 -54.08 -19.136 -48.932Q-21.84 -43.784 -21.84 -36.815999999999995Q-21.84 -31.616 -20.28 -27.351999999999997Q-18.72 -23.087999999999997 -15.86 -20.019999999999996Q-13.0 -16.951999999999998 -8.943999999999999 -15.287999999999998Q-4.887999999999998 -13.623999999999999 0.0 -13.623999999999999Z"/><path transform="translate(760.28 273.55) rotate(48.98)" d="M-36.348 0.0 -7.2280000000000015 -73.42399999999999H7.539999999999999L36.348 0.0H19.083999999999996L-2.9640000000000057 -60.943999999999996H2.9639999999999986L-19.396 0.0ZM-19.812 -13.312V-26.624H20.227999999999994V-13.312Z"/><path transform="translate(810.38 349.36) rotate(64.11)" d="M-8.164000000000001 0.0V-72.384H8.163999999999994V0.0ZM-31.148000000000003 -59.175999999999995V-73.42399999999999H31.14799999999999V-59.175999999999995Z"/><path transform="translate(236.19 822.94) rotate(39.25)" d="M-23.608 0.0V-73.42399999999999H-7.280000000000001V0.0ZM-11.128 -28.808V-42.848H25.583999999999996V-28.808ZM-11.128 -59.384V-73.42399999999999H27.247999999999998V-59.384Z"/><path transform="translate(313.78 873.11) rotate(26.52)" d="M0.2079999999999984 1.248Q-8.112000000000002 1.248 -15.132000000000001 -1.6639999999999997Q-22.152 -4.576 -27.456 -9.776Q-32.76 -14.975999999999999 -35.672 -21.892Q-38.583999999999996 -28.808 -38.583999999999996 -36.815999999999995Q-38.583999999999996 -44.928 -35.672 -51.792Q-32.76 -58.656 -27.56 -63.804Q-22.36 -68.952 -15.34 -71.812Q-8.32 -74.672 0.0 -74.672Q8.216000000000001 -74.672 15.235999999999997 -71.812Q22.255999999999993 -68.952 27.507999999999996 -63.804Q32.76 -58.656 35.672 -51.739999999999995Q38.583999999999996 -44.824 38.583999999999996 -36.711999999999996Q38.583999999999996 -28.703999999999997 35.672 -21.787999999999997Q32.76 -14.872 27.559999999999995 -9.724Q22.359999999999992 -4.576 15.339999999999996 -1.6639999999999997Q8.32 1.248 0.2079999999999984 1.248ZM0.0 -13.623999999999999Q6.552 -13.623999999999999 11.491999999999997 -16.536Q16.431999999999995 -19.448 19.135999999999996 -24.7Q21.839999999999996 -29.951999999999998 21.839999999999996 -36.815999999999995Q21.839999999999996 -42.016 20.279999999999998 -46.227999999999994Q18.72 -50.44 15.808 -53.507999999999996Q12.896 -56.576 8.892 -58.188Q4.887999999999998 -59.8 0.0 -59.8Q-6.552 -59.8 -11.491999999999999 -56.94Q-16.432 -54.08 -19.136 -48.932Q-21.84 -43.784 -21.84 -36.815999999999995Q-21.84 -31.616 -20.28 -27.351999999999997Q-18.72 -23.087999999999997 -15.86 -20.019999999999996Q-13.0 -16.951999999999998 -8.943999999999999 -15.287999999999998Q-4.887999999999998 -13.623999999999999 0.0 -13.623999999999999Z"/><path transform="translate(406.19 906.31) rotate(13)" d="M0.1039999999999992 1.144Q-8.943999999999999 1.144 -15.859999999999998 -2.7039999999999997Q-22.775999999999996 -6.552 -26.675999999999995 -13.363999999999999Q-30.575999999999997 -20.176 -30.575999999999997 -28.912V-73.42399999999999H-14.143999999999998V-27.976Q-14.143999999999998 -23.608 -12.271999999999998 -20.384Q-10.399999999999999 -17.16 -7.123999999999999 -15.443999999999999Q-3.847999999999999 -13.728 0.1039999999999992 -13.728Q4.264000000000003 -13.728 7.384 -15.443999999999999Q10.503999999999998 -17.16 12.323999999999998 -20.332Q14.143999999999998 -23.503999999999998 14.143999999999998 -27.872V-73.42399999999999H30.576V-28.808Q30.576 -20.072 26.728 -13.312Q22.880000000000003 -6.552 16.016000000000002 -2.7039999999999997Q9.152000000000001 1.144 0.1039999999999992 1.144Z"/><path transform="translate(500.26 917) rotate(-0.04)" d="M-31.304 0.0V-73.42399999999999H-19.863999999999997L-14.975999999999999 -58.76V0.0ZM19.344 0.0 -23.919999999999998 -55.431999999999995 -19.863999999999997 -73.42399999999999 23.4 -17.992ZM19.344 0.0 14.975999999999999 -14.664V-73.42399999999999H31.303999999999995V0.0Z"/><path transform="translate(596.95 905.57) rotate(-13.44)" d="M-21.112 0.0V-14.351999999999999H-2.911999999999999Q3.7439999999999998 -14.351999999999999 8.736 -17.003999999999998Q13.728000000000002 -19.656 16.432 -24.752Q19.135999999999996 -29.848 19.135999999999996 -36.815999999999995Q19.135999999999996 -43.784 16.38 -48.775999999999996Q13.624000000000002 -53.768 8.684000000000001 -56.471999999999994Q3.7439999999999998 -59.175999999999995 -2.911999999999999 -59.175999999999995H-21.631999999999998V-73.42399999999999H-2.7040000000000006Q5.616 -73.42399999999999 12.636 -70.77199999999999Q19.656 -68.11999999999999 24.907999999999998 -63.17999999999999Q30.159999999999997 -58.239999999999995 33.019999999999996 -51.532Q35.879999999999995 -44.824 35.879999999999995 -36.711999999999996Q35.879999999999995 -28.703999999999997 33.019999999999996 -21.944Q30.159999999999997 -15.184 24.959999999999997 -10.296Q19.759999999999998 -5.4079999999999995 12.739999999999998 -2.7039999999999997Q5.719999999999999 0.0 -2.496000000000002 0.0ZM-32.135999999999996 0.0V-73.42399999999999H-15.808V0.0Z"/><path transform="translate(684.35 874.04) rotate(-26.24)" d="M-14.351999999999997 -30.264V-42.327999999999996H1.1440000000000055Q6.032000000000004 -42.327999999999996 8.684000000000001 -44.824Q11.335999999999999 -47.32 11.335999999999999 -51.583999999999996Q11.335999999999999 -55.535999999999994 8.736 -58.135999999999996Q6.136000000000003 -60.736 1.2480000000000047 -60.736H-14.351999999999997V-73.42399999999999H3.1200000000000045Q10.399999999999999 -73.42399999999999 15.911999999999999 -70.66799999999999Q21.424 -67.91199999999999 24.544 -63.023999999999994Q27.664 -58.135999999999996 27.664 -51.791999999999994Q27.664 -45.344 24.544 -40.507999999999996Q21.424 -35.672 15.808 -32.967999999999996Q10.192 -30.264 2.6000000000000014 -30.264ZM-26.831999999999997 0.0V-73.42399999999999H-10.503999999999998V0.0ZM13.104 0.0 -9.775999999999996 -31.616 5.200000000000003 -35.672 32.44800000000001 0.0Z"/><path transform="translate(759.89 826.11) rotate(-38.55)" d="M-6.084 -27.352 -34.372 -73.42399999999999H-15.443999999999999L6.292000000000002 -36.192H-5.875999999999998L15.86 -73.42399999999999H34.37200000000001L5.876000000000005 -27.352ZM-8.059999999999999 0.0V-34.839999999999996H8.268V0.0Z"/></g><circle cx="119" cy="500" r="13" fill="#9b9b96"/><circle cx="881" cy="500" r="13" fill="#9b9b96"/><g transform="translate(283.5 249.86) scale(0.42) rotate(16.0 509.5 607.5)"><mask id="udark" maskUnits="userSpaceOnUse" x="0" y="0" width="1060" height="1220"><rect width="1060" height="1220" fill="#fff"/><path d="M135 485C138.67 491.5 151.5 511 157 524C162.5 537 163.17 550 168 563C172.83 576 178.5 589 186 602C193.5 615 203 628 213 641C223 654 233.83 667.67 246 680C258.17 692.33 279.33 709.17 286 715" fill="none" stroke="#000" stroke-width="35.58" stroke-linecap="round" stroke-linejoin="round"/><circle cx="327.88" cy="750.61" r="36.86" fill="none" stroke="#000" stroke-width="30"/><path d="M845 316C846.5 323.5 850.83 346 854 361C857.17 376 861.5 390.83 864 406C866.5 421.17 868.83 436.83 869 452C869.17 467.17 866.83 482 865 497C863.17 512 861.5 526.83 858 542C854.5 557.17 846.33 580.33 844 588" fill="none" stroke="#000" stroke-width="32.39" stroke-linecap="round" stroke-linejoin="round"/><circle cx="815.77" cy="633.22" r="35.23" fill="none" stroke="#000" stroke-width="32"/><path d="M590.6 310.1C561.6 309.2 533.4 310.7 505.1 315.1C476.8 319.5 450.1 325.8 420.6 336.3C391.1 346.8 351.4 362.9 328.2 378C304.9 393.2 282 411.6 280.9 427.5C279.7 443.3 301.1 457.6 321.2 473C341.3 488.3 378.3 502.5 401.5 519.7C424.7 536.9 444.4 555.3 460.6 576.2C476.7 597 485.9 619.9 498.5 644.7C511 669.6 523.3 697.2 535.6 725C548 752.9 562.3 783.2 572.4 811.6C582.4 840 592.4 869.5 595.8 895.7C599.2 921.8 600.2 947.1 592.7 968.5C585.3 989.8 570.3 1006.3 551.1 1023.7C531.8 1041.1 490.3 1057.6 477.2 1072.6C464.1 1087.7 460.3 1103.2 472.5 1113.8C484.7 1124.4 522.5 1133.2 550.5 1136.3C578.5 1139.4 611.4 1137 640.5 1132.4C669.5 1127.8 696.4 1118.4 724.8 1108.7C753.1 1099 785.7 1089.4 810.8 1074.2C835.8 1058.9 869.4 1031.7 875 1017C880.7 1002.4 864.8 993.1 844.5 986.1C824.3 979.2 778.5 983.7 753.6 975.4C728.7 967.1 709 954.2 695 936.1C681.1 918 678.1 892.1 669.9 866.8C661.6 841.6 653.7 812.9 645.6 784.6C637.6 756.3 627.4 726.4 621.5 696.9C615.5 667.4 609.5 636.9 609.8 607.4C610.1 577.8 614.3 547.8 623.1 519.6C631.8 491.3 650.3 462.9 662.3 437.7C674.2 412.4 691.8 387.6 694.6 368C697.4 348.5 696.5 330.1 679.1 320.5C661.8 310.8 619.6 311 590.6 310.1Z" fill="none" stroke="#000" stroke-width="24"/></mask><path d="M263 442.9C263 442.9 307.9 394.9 335 376.5C362.2 358.1 393.8 343.5 425.7 332.6C457.6 321.7 492.7 314.6 526.7 311.2C560.6 307.8 596.3 308.2 629.5 312.3C662.8 316.4 726 335.9 726 335.9C726 336.1 752.9 270.4 748.9 246C745 221.6 723.5 203.1 702.3 189.7C681.2 176.3 650.7 169.4 622.1 165.8C593.4 162.2 561.4 164.8 530.4 168.1C499.4 171.3 466.7 178 436 185.2C405.4 192.4 374.4 200.2 346.4 211.1C318.3 222 290.8 234.3 267.8 250.6C244.8 266.9 220.6 287.7 208.4 308.9C196.2 330.1 185.4 355.7 194.5 378.1C203.6 400.5 263 443.1 263 443.1ZM385 1053C385 1053 432.5 1028.4 449.4 1009.8C466.3 991.2 478 966.4 486.3 941.4C494.5 916.5 498.2 887.3 499 860C499.8 832.7 496.7 803.6 491.2 777.8C485.7 751.9 477 727.4 466.1 704.7C455.1 682 441.4 661.2 425.7 641.5C410.1 621.7 391.9 603.6 372.2 586.2C352.4 568.7 330.3 552.7 307.2 536.9C284.1 521.2 257.1 507.1 233.7 491.6C210.3 476.1 184 461.8 166.8 443.9C149.5 426.1 135.6 406.2 130 384.7C124.5 363.2 127.5 337.3 133.7 315.1C140 292.9 152.8 270 167.4 251.4C182.1 232.9 201.1 217.4 221.7 203.8C242.3 190.1 266.3 179.3 291.1 169.5C316 159.7 343.4 152.1 370.8 144.8C398.2 137.5 427.3 131.4 455.7 125.7C484.1 120.1 513.3 114.2 541.1 110.9C569 107.7 596.9 104.9 622.9 106.2C648.9 107.5 674.3 110.5 697 118.7C719.7 126.8 741.9 139.1 759.2 155.1C776.5 171.1 792.9 192.9 800.9 214.5C808.9 236.1 811.1 261 807.2 284.8C803.2 308.5 789.2 332.9 777.2 357.2C765.3 381.5 748 406 735.5 430.5C722.9 455.1 711 479.7 702.1 504.5C693.1 529.2 686 554.1 681.9 579.2C677.9 604.2 676.1 629.4 678 654.9C679.8 680.3 684.7 706.4 693 731.8C701.3 757.1 713.4 783.7 727.8 807C742.2 830.4 759.9 853.6 779.3 872C798.7 890.3 820.9 907 844.2 917.4C867.5 927.7 919 934 919 934C919.1 934 925.2 881.9 927.9 855.3C930.7 828.7 933.5 801.7 935.7 774.6C937.9 747.4 940 720 941.2 692.5C942.4 665.1 943.3 637.5 943.1 610C942.9 582.5 942.1 554.9 940.1 527.7C938.1 500.4 935.3 473.2 931.1 446.4C926.8 419.6 921.5 392.9 914.6 366.9C907.7 340.8 899.5 314.7 889.6 290.3C879.7 265.8 868.4 241.7 855.2 220C842.1 198.3 827.3 177.6 810.7 160.1C794 142.6 775.7 126.9 755.4 114.8C735.1 102.7 712.5 93.9 688.8 87.7C665.1 81.5 639.3 78.7 613.1 77.4C586.9 76.1 559.2 77.6 531.6 80C504 82.4 475.4 86.7 447.6 91.7C419.7 96.7 391.4 102.6 364.5 110C337.6 117.4 310.8 125.8 286 136.1C261.3 146.4 237.3 158 215.9 171.7C194.6 185.5 174.7 201 157.9 218.7C141.2 236.3 126.6 256.3 115.3 277.5C104 298.8 95.6 322.2 90.3 346.3C85 370.3 83.4 396.4 83.6 421.9C83.8 447.5 87.1 473.9 91.7 499.4C96.2 525 103 550.3 110.9 575.2C118.8 600.1 128.5 624.7 139.1 648.9C149.7 673.2 161.7 697.1 174.3 720.7C186.8 744.4 200.6 767.7 214.4 790.7C228.3 813.7 243 836.4 257.6 858.8C272.1 881.2 287.2 903.3 301.7 925.2C316.2 947 330.9 968.6 344.8 989.9C358.6 1011.2 384.8 1053 384.8 1053Z" fill="#9b9b96" fill-rule="evenodd" mask="url(#udark)"/><path d="M590.6 310.1C561.6 309.2 533.4 310.7 505.1 315.1C476.8 319.5 450.1 325.8 420.6 336.3C391.1 346.8 351.4 362.9 328.2 378C304.9 393.2 282 411.6 280.9 427.5C279.7 443.3 301.1 457.6 321.2 473C341.3 488.3 378.3 502.5 401.5 519.7C424.7 536.9 444.4 555.3 460.6 576.2C476.7 597 485.9 619.9 498.5 644.7C511 669.6 523.3 697.2 535.6 725C548 752.9 562.3 783.2 572.4 811.6C582.4 840 592.4 869.5 595.8 895.7C599.2 921.8 600.2 947.1 592.7 968.5C585.3 989.8 570.3 1006.3 551.1 1023.7C531.8 1041.1 490.3 1057.6 477.2 1072.6C464.1 1087.7 460.3 1103.2 472.5 1113.8C484.7 1124.4 522.5 1133.2 550.5 1136.3C578.5 1139.4 611.4 1137 640.5 1132.4C669.5 1127.8 696.4 1118.4 724.8 1108.7C753.1 1099 785.7 1089.4 810.8 1074.2C835.8 1058.9 869.4 1031.7 875 1017C880.7 1002.4 864.8 993.1 844.5 986.1C824.3 979.2 778.5 983.7 753.6 975.4C728.7 967.1 709 954.2 695 936.1C681.1 918 678.1 892.1 669.9 866.8C661.6 841.6 653.7 812.9 645.6 784.6C637.6 756.3 627.4 726.4 621.5 696.9C615.5 667.4 609.5 636.9 609.8 607.4C610.1 577.8 614.3 547.8 623.1 519.6C631.8 491.3 650.3 462.9 662.3 437.7C674.2 412.4 691.8 387.6 694.6 368C697.4 348.5 696.5 330.1 679.1 320.5C661.8 310.8 619.6 311 590.6 310.1Z" fill="#9b9b96" fill-rule="evenodd"/></g></svg>
```
