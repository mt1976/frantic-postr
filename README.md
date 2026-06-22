# frantic-postr

A Go CLI that connects to Plex, lets you choose a library, reads all collections in that library, and creates collection poster images from a template.

## Run

1. Copy `config/config.example.toml` to `config/config.toml` and update values.
2. Copy `config/config.plex.example.toml` to `config/config.plex.toml` and set your Plex URL/token.
3. Keep `plex_config = "./config.plex.toml"` in `config/config.toml` so secrets stay outside your main config.
4. Ensure your template image exists (`.png` or `.jpg`).
5. Run the explicit poster generation mode:

```bash
go run . -config config/config.toml -gen-posters
```

Running the app without a mode flag prints the help text instead of starting a workflow.

`config/config.plex.toml` is intended to stay local and is ignored by git.

Library selection memory:

- The last selected library (or library set) is remembered.
- On next run, selection prompt shows the previous selection as the default.
- Press Enter to reuse it, or type a new value to replace it.

To also upload each generated poster and set it as that collection's poster in Plex during poster generation, add `-upload-posters`:

```bash
go run . -config config/config.toml -gen-posters -upload-posters
```

To keep request chatter out of the terminal while still writing everything to the log file, add `-quiet`:

```bash
go run . -config config/config.toml -gen-posters -quiet
```

Interactive prompt layout:

- The app uses an old-school terminal screen style for interactive prompts.
- Top rows: app name on the left and current date on the right.
- A separator line follows the header.
- Bottom three rows are reserved for:
  - a separator line,
  - an input row,
  - a feedback row (validation errors, prompt guidance, etc).

## Collection Export / Import / Inject

You can now export collections from one library and import them into another library, including smart collection filter definitions.

1. Export collections from a single selected source library into a JSON file:

```bash
go run . -config config/config.toml -coll-export
```

By default, `-coll-file` values that are just filenames are stored under `output/collections-export/`.
So the default export location is `output/collections-export/collections-export.json`.

1. Import that file into a single selected target library:

```bash
go run . -config config/config.toml -coll-import
```

There is also a compatibility alias for import mode:

```bash
go run . -config config/config.toml -coll-impot
```

To inject smart collections from `config/collections.toml` into a selected library, add `-coll-inject`:

```bash
go run . -config config/config.toml -coll-inject
```

Collection definitions live in `config/collections.toml` and use repeated `[[collection.lookup]]` tables. Put the shared Plex prefix in `base_uri`, then keep each lookup's `content` to just the variable tail, for example `dovi=1` or `push=1&resolution=2.7k&or=1&resolution=4k&pop=1`. The library section id is rewritten automatically when the collection is injected into the selected target library.

## Poster Background Routing

Poster generation can use four template backgrounds:

- `template_image`: default background
- `type_template_image`: used for collections listed in `type_collections_file`
- `studio_template_image`: used for collections listed in `studio_collections_file`
- `admin_template_image`: used for collections listed in `admin_collections_file`

Example config:

```toml
template_image = "../templates/template-3.png"
type_template_image = "../templates/template-1.png"
studio_template_image = "../templates/template-3.png"
admin_template_image = "../templates/template-2.png"
type_collections_file = "./types-collections.txt"
studio_collections_file = "./studio-collections.txt"
admin_collections_file = "./admin-collections.txt"
```

Matching rules for collection names in those files:

- Case-insensitive
- Spaces and punctuation are ignored on both sides before comparison
- If a collection is not in any list, the default background is used
- If a collection appears in multiple lists, precedence is `admin`, then `studio`, then `type`

Poster output details:

- Generated poster filenames use the collection title only.
- A per-library poster CSV report is written under the library output folder and includes the `background` used for each collection.

## Collection Audit / Cleanup

To find duplicate collection names in a selected library and write a CSV report with item counts, use `-coll-dupes`:

```bash
go run . -config config/config.toml -coll-dupes
```

To delete every non-smart collection from a selected library and write a CSV audit, use `-coll-delete-non-smart`:

```bash
go run . -config config/config.toml -coll-delete-non-smart
```

Notes:

- Export/import modes require selecting exactly one library.
- Existing collections in the target library with the same title are skipped.
- For smart collections, section references inside filter URIs are rewritten from source library to target library.

## Library Clone

Clone mode creates a new Plex library from a selected source library by copying:

- Library path mappings (all source `Location` paths)
- Core library setup values (type, agent, scanner, language)
- Library preferences (`/prefs` settings)

Run clone mode:

```bash
go run . -config config/config.toml -clone
```

Flow:

1. Select one source library.
2. Enter a new library name when prompted.
3. Press Enter to accept the default name: `<source-name>-clone`.

Notes:

- Clone mode is exclusive with collection import/export modes.
- `-upload-posters` is not used in clone mode.

## Label Mode

Label mode scans one selected library and adds labels to items whose title contains the `-find` text.

Example:

```bash
go run . -config config/config.toml -label -find abandoned -add urbsex,abandoned
```

You can quote `-find` to include spaces:

```bash
go run . -config config/config.toml -label -find "abandoned house" -add urbsex,abandoned
```

To also update category tags (Plex Genre tags) using the same `-add` values:

```bash
go run . -config config/config.toml -label -find abandoned -add urbsex,abandoned -update-category
```

To update only category tags (and skip label updates):

```bash
go run . -config config/config.toml -label -find abandoned -add urbsex,abandoned -only-category
```

You can also define multiple title lookup rules in config and run `-label` without `-find` / `-add`:

```toml
[[label.lookup]]
title_contains = "abandoned"
labels = ["urbsex", "abandoned"]
categories = ["urbsex", "abandoned"]
update_category = true

[[label.lookup]]
title_contains = "warehouse"
labels = ["urbsex"]
only_category = true

[[label.lookup]]
title_contains_any = ["Chem", "PnP"]
labels = ["Chems"]
categories = ["Chems"]
update_category = true
```

Run with:

```bash
go run . -config config/config.toml -label
```

Behavior:

- Matching is case-insensitive and checks the whole title string using substring matching.
- Examples that match `abandoned`: `.abanDONED.`, `_abandonedHouse_`.
- Labels in `-add` are comma-separated.
- Existing labels are preserved; only missing labels are added.
- With `-update-category`, existing category tags are preserved; only missing category tags are added.
- With `-only-category`, label updates are skipped and only category tags are updated.
- `-label` requires either:

  - `-find` with `-add`, or
  - one or more `[[label.lookup]]` entries in config.

- If title fields are empty, label matching falls back to media file path text.
- `-update-category` and `-only-category` only apply to `-label` mode. In other modes, they are ignored and an error is logged.

## Clean Mode

Clean mode scans one selected library and sanitizes item titles for safer searching.

```bash
go run . -config config/config.toml -clean
```

To translate first, explicitly add `-translate`:

```bash
go run . -config config/config.toml -translate -clean
```

Rules:

- Special characters are replaced with spaces.
- `@` is preserved.
- `&` is replaced with `and`.
- `#` followed by a number is replaced with `No.` (example: `#12` -> `No. 12`).
- Repeated spaces are compressed to a single space.
- First letter is uppercased.
- Blank titles become `Unknown`.

Notes:

- Logs include before/after title values for every changed item.
- Only title is updated; sort title is left unchanged.
- If title and/or sort title are blank, clean mode seeds the blank field(s) from the media filename (without extension) before cleaning.

Custom replacements can be configured to future-proof behavior:

```toml
[clean.replacements]
"&" = " and "
"£" = " gbp "
"$" = " usd "
"FULL MOVIE" = " "
"cum#" = " climax number "
```

These replacements are applied before the built-in clean rules.

Translation is feature-flagged and runs only when `-translate` is provided.

Translate-only mode (no cleaning):

```bash
go run . -config config/config.toml -translate
```

You can still configure the translation endpoint/API key:

```toml
[clean]
translate_api_http_address = "https://libretranslate.com/translate"
translate_endpoint = "https://libretranslate.com/translate"
translate_api_key = ""
translate_rate_limit_per_minute = 10
```

When `-translate` is used with `-clean`, titles are translated to English first, then sanitized.
Translation requests are throttled by `translate_rate_limit_per_minute` to help avoid `429 Too Many Requests` responses.

The app writes posters to `output/<library-name>/` (or `output_dir/<library-name>/`) and logs startup, config reads, Plex calls, processing results, and file creation details to stdout and a timestamped log file derived from `log_file` for that run. Plex API requests are also logged as executable `curl` commands so the request can be replayed manually from a shell.

## Collection Path Clean Mode

Path clean mode scans one selected library, then lets you choose a collection by typing at least the first three characters of its name.

```bash
go run . -config config/config.toml -coll-path-clean
```

Flow:

1. Select one library, using the last remembered library choice as the default when available.
2. Enter at least the first three characters of a collection name.
3. The app shows all collections whose names start with those characters.
4. If there is only one match, or an exact match, the app asks for confirmation.
5. If there are multiple matches, select one from the list and confirm before continuing.
6. If you answer `N`, the process stops.
7. If you answer `Y`, the app scans every item in the collection and rebuilds each title from the file path only.

Title format:

- The filename stem becomes the leading part of the title.
- Each parent directory is appended in reverse order using ` - ` as the separator.
- Windows paths are handled correctly even when the app runs on macOS.

Example:

- `V:\FILTH\TORRENT\chalate2000\0gnfqk81vt7lmlg8b4ykb_source.mp4`
- `0gnfqk81vt7lmlg8b4ykb source - chalate2000 - TORRENT - FILTH`

Notes:

- The mode respects `-quiet` and `-trial`/`-trail` the same way the other write modes do.
- Each update is recorded in `output/path-clean/` as a CSV audit.
- `-coll-path-clean` is exclusive with the other single-mode workflows.

Startup validation is strict: if `plex.base_url` / `plex.token` are missing in both `config/config.toml` and `plex_config`, or required paths like `template_image` / `output_dir` do not exist, the app logs an error and exits.

## Logging And Retries

Logs now include level tags and color output in terminal:

- `INFO`
- `SUCCESS` (green)
- `WARNING` (yellow)
- `ERROR` (red)
- `API` (cyan)
- `MATCH` (used in label mode to highlight matched find text)

Disable color output when needed (CI/log parsers/plain terminals):

```bash
go run . -config config/config.toml -no-color
```

Timeout retries are configurable in `[plex]`:

```toml
[plex]
retries = 3
```

When a network operation fails with timeout conditions like `context deadline exceeded`, the app retries up to `plex.retries` times.

## Trail Mode

Use `-trail` to process as normal but skip all Plex write operations (`PUT`/`POST`).

```bash
go run . -config config/config.toml -trail
```

This works across all modes and logs each skipped write as a warning.
