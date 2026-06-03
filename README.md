# frantic-postr

A Go CLI that connects to Plex, lets you choose a library, reads all collections in that library, and creates collection poster images from a template.

## Run

1. Copy `config.example.json` to `config.json` and update values.
2. Ensure your template image exists (`.png` or `.jpg`).
3. Run:

```bash
go run . -config config.json
```

The app writes posters to `output/<library-name>/` (or `output_dir/<library-name>/`) and logs startup, config reads, Plex calls, processing results, and file creation details to stdout and the configured log file.
