# frantic-postr

A Go CLI that connects to Plex, lets you choose a library, reads all collections in that library, and creates collection poster images from a template.

## Run

1. Copy `config.example.toml` to `config.toml` and update values.
2. Ensure your template image exists (`.png` or `.jpg`).
3. Run:

```bash
go run . -config config.toml
```

To also upload each generated poster and set it as that collection's poster in Plex during processing, add `-upload`:

```bash
go run . -config config.toml -upload
```

The app writes posters to `output/<library-name>/` (or `output_dir/<library-name>/`) and logs startup, config reads, Plex calls, processing results, and file creation details to stdout and a timestamped log file derived from `log_file` for that run. Plex API requests are also logged as executable `curl` commands so the request can be replayed manually from a shell.

Startup validation is strict: if `plex.base_url` / `plex.token` are missing, or required paths like `template_image` / `output_dir` do not exist, the app logs an error and exits.
