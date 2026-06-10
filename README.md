# frantic-postr

A Go CLI that connects to Plex, lets you choose a library, reads all collections in that library, and creates collection poster images from a template.

## Run

1. Copy `config.example.toml` to `config.toml` and update values.
2. Ensure your template image exists (`.png` or `.jpg`).
3. Run the explicit poster generation mode:

```bash
go run . -config config.toml -gen-posters
```

Running the app without a mode flag prints the help text instead of starting a workflow.

Library selection memory:

- The last selected library (or library set) is remembered.
- On next run, selection prompt shows the previous selection as the default.
- Press Enter to reuse it, or type a new value to replace it.

To also upload each generated poster and set it as that collection's poster in Plex during poster generation, add `-upload-posters`:

```bash
go run . -config config.toml -gen-posters -upload-posters
```

To keep request chatter out of the terminal while still writing everything to the log file, add `-quiet`:

```bash
go run . -config config.toml -gen-posters -quiet
```

## Collection Export / Import / Inject

You can now export collections from one library and import them into another library, including smart collection filter definitions.

1. Export collections from a single selected source library into a JSON file:

```bash
go run . -config config.toml -coll-export -coll-file collections-export.json
```

2. Import that file into a single selected target library:

```bash
go run . -config config.toml -coll-import -coll-file collections-export.json
```

There is also a compatibility alias for import mode:

```bash
go run . -config config.toml -coll-impot -coll-file collections-export.json
```

To inject smart collections from `collections.toml` into a selected library, add `-coll-inject`:

```bash
go run . -config config.toml -coll-inject
```

Collection definitions live in `collections.toml` and use repeated `[[collection.lookup]]` tables. Put the shared Plex prefix in `base_uri`, then keep each lookup's `content` to just the variable tail, for example `dovi=1` or `push=1&resolution=2.7k&or=1&resolution=4k&pop=1`. The library section id is rewritten automatically when the collection is injected into the selected target library.

## Collection Audit / Cleanup

To find duplicate collection names in a selected library and write a CSV report with item counts, use `-coll-dupes`:

```bash
go run . -config config.toml -coll-dupes
```

To delete every non-smart collection from a selected library and write a CSV audit, use `-coll-delete-non-smart`:

```bash
go run . -config config.toml -coll-delete-non-smart
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
go run . -config config.toml -clone
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
go run . -config config.toml -label -find abandoned -add urbsex,abandoned
```

You can quote `-find` to include spaces:

```bash
go run . -config config.toml -label -find "abandoned house" -add urbsex,abandoned
```

To also update category tags (Plex Genre tags) using the same `-add` values:

```bash
go run . -config config.toml -label -find abandoned -add urbsex,abandoned -update-category
```

To update only category tags (and skip label updates):

```bash
go run . -config config.toml -label -find abandoned -add urbsex,abandoned -only-category
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
go run . -config config.toml -label
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
go run . -config config.toml -clean
```

To translate first, explicitly add `-translate`:

```bash
go run . -config config.toml -translate -clean
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
go run . -config config.toml -translate
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

Startup validation is strict: if `plex.base_url` / `plex.token` are missing, or required paths like `template_image` / `output_dir` do not exist, the app logs an error and exits.

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
go run . -config config.toml -no-color
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
go run . -config config.toml -trail
```

This works across all modes and logs each skipped write as a warning.
