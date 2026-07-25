# frantic-postr Workflows (User Stories)

This document captures the primary user workflows in user-story form.

## Configuration and Access

### Story 1: Configure Plex and runtime settings
As a media admin, I want to configure Plex connectivity and runtime values in one place so that all workflows run with valid credentials and predictable behavior.

Acceptance notes:
- Plex URL/token, retry, and worker settings are editable.
- Config save applies all Config tab cards.
- Save button appears only on the Config tab.

### Story 2: Verify Plex connectivity before heavy runs
As an operator, I want to test Plex connection before running workflows so that failures are caught early.

Acceptance notes:
- Test Plex connection runs without launching full workflows.
- The button reports loading/success/error states.
- Toast/banner feedback communicates the result.

### Story 3: Manage hidden config file content safely
As a cautious operator, I want to edit key config/list content via dialogs instead of path fields so that sensitive paths stay hidden and errors are reduced.

Acceptance notes:
- Type/studio/admin collection lists are editable via modal.
- Label and collection config content is editable via modal.
- Supplemental Plex config path is not exposed as a plain UI field.

## Poster Workflows

### Story 4: Generate posters for selected libraries
As a content curator, I want to generate posters for selected libraries so that collections get consistent visuals.

Acceptance notes:
- Multi-select library sections.
- Optional upload to Plex.
- Optional label-type behavior.
- Optional missing-only mode.
- Optional dry-run mode.

### Story 5: Tune poster style before full run
As a designer/operator, I want on-demand template preview rendering so that I can validate text placement and styling quickly.

Acceptance notes:
- Choose template kind (default/type/studio/admin).
- Enter sample text.
- Preview reflects font, color, shadow, glow, and Y-offset settings.

## Text Normalization and Translation

### Story 6: Clean titles in a controlled way
As a librarian, I want to clean titles with configurable replacements so that naming stays standardized.

Acceptance notes:
- Clean workflow can run by selected library.
- Replacements support modern editor and raw fallback.
- Translate-to-English toggle can influence clean config behavior.

### Story 7: Translate titles separately
As a multilingual operator, I want a dedicated translation workflow so that I can run language normalization independently from other operations.

Acceptance notes:
- Translation runs by selected library.
- Dry-run option available.
- Translate endpoint/key/rate limit are configurable in Config.

## Stats, Labels, and Library Utilities

### Story 8: Run stats with exclusions
As an analyst, I want to run stats while excluding noise words so that reports are actionable.

Acceptance notes:
- Exclude words editable via chips/list and raw fallback.
- Run stats is a primary action in UI hierarchy.

### Story 9: Apply labels based on find criteria
As a taxonomy manager, I want to find matching media and apply labels/categories so that browsing and downstream automation improve.

Acceptance notes:
- Supports find text, label list, category controls, and dry-run.

### Story 10: Clone a library
As a Plex maintainer, I want to clone a library to a new name so that I can test or stage changes safely.

Acceptance notes:
- Source section selection and target name input are required.

## Collections Lifecycle

### Story 11: Export and import collections
As a collections manager, I want to export from one library and import into another so that I can migrate or duplicate collection setups.

Acceptance notes:
- Export writes JSON artifacts under output export location.
- Import supports choosing existing file and uploading a new one.
- Import action supports dry-run.

### Story 12: Inject configured smart collections
As an automation owner, I want to inject predefined smart collections so that baseline collection sets can be re-applied quickly.

Acceptance notes:
- Uses configured collection definitions.
- Supports dry-run.

### Story 13: Audit and clean collection quality
As an operator, I want duplicate audits, non-smart deletion, and path-clean tools so that collection hygiene stays high.

Acceptance notes:
- Duplicate audit runs as a primary action.
- Non-smart delete supports dry-run safeguards.
- Path-clean targets a selected library + collection.

## Protection and Recovery

### Story 14: Create backups before major changes
As a risk-aware admin, I want one-click backup creation so that I can recover from mistakes.

Acceptance notes:
- Backup includes key config/template/font/state assets.
- Retention is controlled by config.

### Story 15: Restore and rollback with confidence
As an operator, I want restore and rollback workflows so that I can recover from bad configuration or content changes.

Acceptance notes:
- Restore supports filter-based target selection and dry-run.
- Rollback restores pre-restore state when available.
- Restore/rollback actions emit report artifacts.

## Runtime and Observability

### Story 16: Monitor long-running actions across tabs
As an operator, I want progress and logs to remain visible while I navigate so that I can supervise operations without losing context.

Acceptance notes:
- Persistent progress card and operation log.
- Stop active process when supported.
- Download output artifact when available.

## UI Presentation and Interaction Rules

### Story 17: Use a clear, consistent control room layout
As a user, I want a consistent card-and-tab layout so that navigation and actions are predictable.

Acceptance notes:
- Two cards per row on larger screens.
- Single card spans full row when alone in final row.
- Cards in same row visually share tallest height.
- Action buttons usually appear in dedicated action rows.
- Explicit button groups remain inline on one row.

### Story 18: Keep secrets safer in UI
As a security-conscious user, I want token/key fields masked with reveal controls so that sensitive values are less exposed during routine use.

Acceptance notes:
- API token and translate key are password-style inputs.
- Reveal/hide toggles are available.

## Operational Definition of Done for Workflow Changes

A workflow change is complete when:
- CLI/web entry points still work for affected flows.
- Relevant config save/load behavior still round-trips.
- Progress/log feedback still functions.
- Any affected story acceptance notes above still hold.
- `go test ./...` passes.
