# rren

`rren` (**r**egex **ren**ame) is a command-line tool that renames files and
folders using regular expressions defined in a YAML configuration file.
Multiple rules are applied in order, so the output of one rule feeds the next.

---

## Features

- Apply a chain of regex rules to file names, folder names, or both.
- Preview all renames in a table before anything is changed.
- Skipped entries (no rule matched) are shown as `-- Skipped --`.
- Interactive confirmation prompt (can be bypassed with `--yes`).
- Dry-run mode for safe previewing.
- Recursive mode to process sub-directories.
- Per-rule `apply_to` targeting: `files`, `folders`, or `both`.
- Optional per-rule description for self-documenting configs.

---

## Installation

### From source

```bash
git clone https://github.com/giulianozor/rren.git
cd rren
make install          # installs to /usr/local/bin by default
# or
make install PREFIX=~/.local
```

### Build only (no install)

```bash
make build            # produces ./rren
```

---

## Usage

```
rren [flags]

Flags:
  -n, --dry-run          Preview renames without applying any changes
  -c, --config string    Path to the configuration file (default ~/.config/rren.yaml)
  -r, --recursive        Rename files and folders recursively
  -y, --yes              Skip the confirmation prompt
  -p, --path string      Root path to process (default ".")
  -h, --help             Show help
```

---

## Configuration file

The default config path is `~/.config/rren.yaml`. Override with `--config`.

```yaml
rules:
  - description: "Optional human-readable description"
    pattern: '<Go regular expression>'
    replacement: '<replacement string>'   # $1, ${1}, $name capture groups
    apply_to: files                       # files | folders | both  (default: both)
```

> **Note:** `pattern` and `replacement` are required for every rule.  
> Capture groups use Go's `$1` / `${1}` syntax in the replacement string.

---

## Examples

### TV show episode normalisation

Rename files like `show.name.01x02.720p.mkv` → `S01E02.mkv`.

```yaml
rules:
  - description: "TV show episode normalisation"
    pattern: '.*(\d\d)[exEX](\d+).*\.(\w+)'
    replacement: 'S${1}E${2}.${3}'
    apply_to: files
```

**Before / After table:**

```
+------+-----------------------------+--------------+
| Type | Source                      | Destination  |
+------+-----------------------------+--------------+
| file | show.name.01x02.720p.mkv    | S01E02.mkv   |
| file | another.show.02E05.1080p.mp4| S02E05.mp4   |
| file | not_an_episode.txt          | -- Skipped --|
+------+-----------------------------+--------------+
```

### Movie year-first rename

Rename files like `Movie Title (2023) BluRay.mkv` → `2023 - Movie Title.mkv`.

```yaml
rules:
  - description: "Movie year-first rename"
    pattern: '(.*)\s\((\d{4})\).*(\.m..)'
    replacement: '$2 - $1$3'
    apply_to: files
```

**Before / After table:**

```
+------+----------------------------------+-----------------------------+
| Type | Source                           | Destination                 |
+------+----------------------------------+-----------------------------+
| file | Inception (2010) BluRay.mkv      | 2010 - Inception.mkv        |
| file | The Dark Knight (2008) 1080p.mp4 | 2008 - The Dark Knight.mp4  |
| file | notes.txt                        | -- Skipped --               |
+------+----------------------------------+-----------------------------+
```

### Combining rules (sample config)

Use the provided `config.example.yaml` which combines both rules above.
Rules are applied in order — the output of each rule becomes the input to the next.

```bash
rren --config config.example.yaml --path ~/Videos --dry-run
```

### Dry-run preview

```bash
rren --dry-run --path ~/Videos --recursive
```

### Rename current directory non-recursively, skipping confirmation

```bash
rren --yes
```

### Use a custom config file

```bash
rren --config ~/my-rules.yaml --path /data/media
```

---

## Summary output

After a rename run, `rren` prints a summary:

```
=== Summary ===
  Renamed : 14
  Skipped : 3
```

In dry-run mode the header changes to:

```
=== Dry-run summary (no files were changed) ===
  Renamed : 14
  Skipped : 3
```

---

## Makefile targets

| Target           | Description                                   |
|------------------|-----------------------------------------------|
| `make`           | Build the binary (alias for `make build`)     |
| `make build`     | Compile `rren` into the current directory     |
| `make install`   | Build and install to `$(PREFIX)/bin`          |
| `make uninstall` | Remove the installed binary                   |
| `make clean`     | Remove build artefacts                        |
| `make test`      | Run all tests                                 |
| `make lint`      | Run `go vet`                                  |
| `make help`      | Print available targets                       |

Set `PREFIX` to change the install location (default `/usr/local`):

```bash
make install PREFIX=~/.local
```
