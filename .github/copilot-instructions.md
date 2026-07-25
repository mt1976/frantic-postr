<!-- mermaid-ai-skills:start -->
## Mermaid Diagrams

When the user asks to create, edit, or visualize a diagram, follow the
instructions in `.github/instructions/mermaid.instructions.md`.
<!-- mermaid-ai-skills:end -->

## App Regeneration Contract (frantic-postr)

Use this section when implementing, refactoring, or regenerating the app so another AI can recreate the same product experience and functionality.

### Product Identity

- Build and maintain a Go application with:
	- CLI workflows for Plex/media operations.
	- A local UIKit-based web control room for the same workflows.
- Keep behavior and naming aligned with the current modes and UI labels.
- Preserve the repository intent: practical media-library operations, strong config ergonomics, and operation traceability.

### Core Architecture

- Keep Go as the implementation language.
- Preserve the current layered structure:
	- `main.go` for CLI bootstrap and mode dispatch.
	- `app/core` for workflow/business logic.
	- `app/reports` for reporting/export CSV/JSON helpers.
	- `app/web` for web server, API handlers, template rendering, and UI state endpoints.
- Prefer server-rendered HTML templates (embedded in Go templates) with vanilla JS + UIKit classes.
- Do not replace the app with SPA frameworks unless explicitly requested.

### Web UI Visual System

- Keep the current visual language:
	- Warm paper-like background, rounded cards, subtle shadows.
	- Expressive heading typography and mono accents for technical meta.
	- UIKit components with custom CSS tokens and gradients.
- Buttons:
	- Primary actions use a subtle warm gradient and slightly taller hit area.
	- Secondary/danger/default semantics must stay meaningful and consistent.
- Card layout:
	- Config/workflow pages use a shared 2-column grid.
	- On mobile widths, collapse to one column.
	- If a final row has one card, it spans full row width.
	- Cards in a row should visually match the tallest card in that row.

### Config Tab Composition (Required)

Maintain these cards and intent in the Config tab:

- Configuration:
	- Output/log/retention and general non-secret app config fields.
- Plex Configuration:
	- Plex URL/token/retry/workers settings.
	- Include a Test Plex connection button.
- Config Files and Templates:
	- Template image selectors and uploads (default/type/studio/admin).
	- Hidden path fields and editor-launch buttons for supported config files/lists.
- Poster Configuration:
	- Template preview generation controls.
	- Font/color/shadow/glow/y-offset settings.
- Text Cleanup and Stats:
	- Translation endpoint/API key/rate limit.
	- Translate-to-English toggle for clean config.
	- Clean replacements modern editor with raw fallback.
	- Stats exclude words modern editor with raw fallback.

### Tab Bar and Save Behavior

- Keep tab navigation for: Config, Runtime, Posters, Library, Collections, Backup & Restore.
- Keep a global Save config button aligned to the tab bar right side.
- Save config button visibility rule:
	- Visible only when Config tab is active.
- Save action writes all relevant config values currently represented on the Config screen.

### Action-Row and Button Layout Rules

- In forms, action buttons should generally appear in their own action row.
- Exception: explicit button groups (for example stop/download utility controls) stay on one shared row.
- Do not force action-row buttons to stretch full width unless explicitly requested.
- Keep critical action hierarchy consistent:
	- `Run stats`, `Export collections`, and `Audit duplicates` are primary actions.
	- `Test Plex connection` is a secondary action.

### Sensitive Data / Config Editing Rules

- Hide file-system path inputs for sensitive/supplemental config references in web mode.
- Provide modal-based editing for supported config/list content instead of exposing raw path fields.
- Keep Plex supplemental config path hidden from direct web editing.
- Token/API key fields should be password type with reveal/hide controls.

### Runtime and Operation UX

- Keep a persistent operation progress area independent of active tab.
- Keep operation log accessible, copyable, and sanitized for control/ANSI noise.
- Preserve stop-action behavior and downloadable output file affordances.
- Keep user feedback style:
	- Banner + toast style notifications for success/error states.

### Workflow Coverage (Must Remain)

Maintain web and CLI support for at least these workflows:

- Poster generation (with optional upload/label-type/missing-only/trial toggles).
- Clean titles.
- Translate titles.
- Stats.
- Label operations.
- Clone library.
- Collection export/import/inject.
- Duplicate audits.
- Delete non-smart collections.
- Path clean collection.
- Backup, restore, rollback.

### Config and Persistence Expectations

- Continue supporting default config behavior when `-config` is omitted.
- Preserve split Plex secret strategy where secrets can live in supplemental Plex config.
- Preserve backup/restore artifact generation and operation reporting conventions.
- Preserve remembered library selection behavior.

### API and Server Behavior

- Keep local web API shape stable for current frontend behaviors.
- Continue supporting endpoints needed for:
	- Runtime state/sections/options loading.
	- Config load/save.
	- Config content modal read/write flows.
	- File upload/list for template and import assets.
	- Template preview generation.
	- Plex test action.
	- Workflow/action invocation + progress polling + stop.

### Regeneration Quality Bar

When asked to regenerate the app:

- Reproduce feature parity before cosmetic reinvention.
- Keep control IDs, endpoint contracts, and payload keys stable unless migration is explicitly requested.
- Keep all existing tabs/cards/workflows operational.
- Keep mobile responsiveness and desktop density balanced.
- After edits, run tests and fix regressions before finalizing.

### Quick-Start Regeneration Checklist

Use this order to rebuild the app with minimal drift:

1. Scaffold the Go app structure (`main.go`, `app/core`, `app/reports`, `app/web`) and wire CLI mode dispatch.
2. Implement core workflows first (posters, clean, translate, stats, labels, collections, backup/restore/rollback).
3. Add web server routes and JSON APIs for config/state/options/action execution/progress/stop.
4. Build the UIKit dashboard shell with tab navigation and persistent operation progress/log area.
5. Recreate Config tab cards exactly: Configuration, Plex Configuration, Config Files and Templates, Poster Configuration, Text Cleanup and Stats.
6. Add sensitive-config behaviors: hidden path fields, modal content editors, password reveal controls.
7. Add template/media helpers: template image upload/select, import file upload/select, template preview generation.
8. Apply required layout/style rules: 2-column cards, lone-card full-row span, equal-height rows, primary-gradient button style.
9. Enforce interaction rules: Save config visible only on Config tab, action-row/button-group behavior, toast/banner feedback.
10. Run `go test ./...`, verify no diagnostics, and validate key UI workflows end-to-end before finalizing.

### Definition of Done (DoD)

A regeneration/refactor task is done only when all items below are satisfied:

1. Build and tests pass with `go test ./...`.
2. CLI modes still execute with expected flags and no breaking flag-name drift.
3. Web tabs, cards, and actions are all reachable and functional.
4. Config save/test flows work end-to-end (including hidden-path/modal editors).
5. Operation progress/log/stop/download behaviors remain intact.
6. Visual rules are preserved (2-column layout, card height parity, button semantics).
7. Security/ergonomic rules are preserved (password fields + reveal, hidden supplemental paths).
8. No new diagnostics are introduced in edited files.
9. Key IDs, API payload keys, and endpoint contracts are unchanged unless explicitly requested.
10. User-visible behavior changes are documented in PR notes or task summary.

### Functional Descriptions (Key Capabilities)

- Configuration management:
	- Load current runtime-config values from server.
	- Edit supported values in cards; save writes back to active TOML sources.
	- Use scoped modal editors for collection lists and config file contents.
- Plex connectivity:
	- Test connection using current Plex fields.
	- Provide inline visual test state and toast/banner feedback.
- Poster generation:
	- Select library sections and run generation workflow.
	- Optional toggles for upload/label/missing-only/dry-run behavior.
- Text cleanup and translation:
	- Run clean/translate actions by library.
	- Manage replacements and exclude-word lists with modern editor + raw fallback.
- Collection management:
	- Export, import, inject, duplicate audit, non-smart delete, and path-clean.
- Backup lifecycle:
	- Create backup, restore backup, rollback most recent restore.
- Runtime observability:
	- Persistent cross-tab operation progress and logs.
	- Copy logs and stop active process when supported.

### Presentation Rules (UI Construction)

- Use UIKit layout primitives with custom CSS only where needed.
- Keep tabs in this order: Config, Runtime, Posters, Library, Collections, Backup & Restore.
- Keep all workflow content inside cards.
- Card grid rules:
	- 2 columns desktop/tablet.
	- 1 column on narrow screens.
	- If final row has one card, span full row.
	- Cards in same row should match tallest card height.
- Button rules:
	- Primary buttons use gradient and slightly larger vertical size.
	- Action buttons typically live on their own row.
	- Explicit action groups remain inline on one row.
	- Action-row buttons wrap content, do not stretch full width unless asked.
- Save config button:
	- Aligned at tab-bar level (right side).
	- Visible only when Config tab is active.

### Pseudocode Playbooks

#### 1) Config Save

```text
onSaveConfig():
	payload.general = readConfigFormFields()
	payload.general.translate_to_english = readToggle("translate-to-english")
	payload.general.template_fields = readTemplateAndFontFields()
	payload.paths = readHiddenPathBindings()
	POST /api/config with payload
	if success: show success toast + refresh state
	else: show error banner/toast
```

#### 2) Tab-Aware Save Button Visibility

```text
updateSaveVisibility():
	activePanel = current uk-switcher panel
	if activePanel contains #config-form:
		show #save-config-global
	else:
		hide #save-config-global
```

#### 3) Operation Runner

```text
runAction(mode, args):
	POST /api/action/run { mode, args }
	mark progress UI active
	poll /api/action/status until terminal state
	update log/progress each poll
	enable output download when file provided
```

#### 4) Scoped Config Content Editing

```text
openConfigEditor(scope):
	GET /api/config/content?scope=<scope>
	if scope is list-type:
		render list/chip editor + raw fallback
	else if scope is structured TOML:
		render structured editor + raw fallback
	save => POST /api/config/content { scope, content }
```

### External Libraries and Why They Exist

- Go standard library:
	- HTTP server, templates, JSON, filesystem, image primitives, and process-safe utilities.
- UIKit (CDN CSS/JS):
	- Rapid, consistent component styling and behaviors (tabs, modals, accordions).
- Google Fonts (Space Grotesk + IBM Plex Mono):
	- Preserve the intended visual identity and technical readability.
- `golang.org/x/image/font` and related font packages:
	- Server-side text rendering for poster/template preview generation.

Do not replace these libraries unless explicitly requested.

### User Workflow Notes (Implementation Targets)

- A user starts in Config to verify Plex, templates, and cleanup settings.
- User saves config once, then moves to workflow tabs to run operations.
- While operations run, user can switch tabs and still monitor progress/logs.
- User can stop long-running actions and download generated outputs.
- User can recover from mistakes via backup/restore/rollback workflows.
