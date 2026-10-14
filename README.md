# discscan

[![CI](https://github.com/gsaranov/discscan/actions/workflows/ci.yml/badge.svg)](https://github.com/gsaranov/discscan/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/gsaranov/discscan.svg)](https://pkg.go.dev/github.com/gsaranov/discscan)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**See what is eating your disk, right from the terminal.**

discscan is a fast, dependency-free disk usage analyzer written in Go. Pick a
drive from an interactive menu and it shows you:

- **Largest files**: the biggest files on the drive.
- **Recent large files**: big files that appeared in the last hours, days or
  weeks, so you can see what filled the drive lately.
- **Largest folders**: where the space goes, with subfolder totals.
- **Programs & games**: installed programs, games and app data folders ranked
  by total size. Folders full of thousands of small files show up here too.

Every result comes with a **cleanup hint**: `delete`, `cache`, `move`,
`system` or `careful`. The hint says what the item is and how to get the
space back safely, for example "page file: move to another drive…", "crash
dump", "Unreal Engine derived data cache, rebuilt automatically" or "installer
in Downloads".

```
  discscan v1.0.0 — see what is eating your disk

  1  Largest files        the biggest files on a drive
  2  Recent large files   what filled the drive lately
  3  Largest folders      where the space goes
  4  Programs & games     apps, games and app data, all files counted
  q  Quit

Choose mode [1]: 2

   1  C:\    Windows 11     Fixed     █████████░░░  75%  115.50 GB free of 464.71 GB
   2  D:\    Data           Fixed     ██████░░░░░░  46%  504.28 GB free of 931.51 GB
   p  Enter a folder path
   b  Back

Choose drive [1]: 1
Minimum file size (e.g. 500MB, 1GB) [100MB]: 1GB
Appeared within (e.g. 24h, 3d, 2w) [7d]:
Maximum files to show [100]:

   #        SIZE  CHANGED                 AGE  PATH
   1     4.87 GB  2026-10-14 09:12     2h ago  C:\Users\me\.ollama\models\blobs\sha256-a3de86cd…
      ↳ careful Ollama model: remove with 'ollama rm <name>' or relocate with OLLAMA_MODELS
   2    14.15 GB  2026-10-12 21:40     1d ago  C:\Users\me\Downloads\dataset.zip
      ↳ delete  archive in Downloads: delete once unpacked or installed, or move to another drive
   3     2.31 GB  2026-10-09 18:05     4d ago  C:\Users\me\AppData\Local\Temp\installer.msi
      ↳ delete  temporary file: safe to delete when the program that made it is closed

Total: 21.33 GB in 3 files

Hints: delete 16.46 GB · careful 4.87 GB

Scanned 1,284,113 files in 402,551 folders (348.20 GB) under C:\ in 21.4s

[1-3] show in Explorer · [s] save CSV · [Enter] menu:
```

## Features

- **Fast.** Directories are read in parallel (2 × CPU cores by default). About
  850,000 files take a few seconds on an SSD.
- **Constant memory.** Only the top N results are kept (bounded heap), so
  scanning millions of files needs very little RAM.
- **Interactive or scriptable.** Use the menu, or call subcommands with flags
  from scripts and scheduled tasks.
- **Finds downloads and copies.** On Windows, "recent" uses the later of the
  modification and creation times. Copied, unpacked and downloaded files keep
  their original modification date but get a fresh creation date, so they still
  show up.
- **Programs & games.** Each app is a direct child of a known location:
  `Program Files`, `ProgramData`, `AppData\Local|Roaming|LocalLow`,
  `AppData\Local\Programs`, Steam/Epic/GOG/Xbox/EA/Ubisoft/Riot libraries, a
  top-level folder of the drive, or a dot-folder in your profile (`.ollama`,
  `.gradle`, …). Nested apps are counted once, so Steam games are not added to
  Steam's own size.
- **Cleanup hints.** About 40 built-in rules recognize the page file,
  hibernation file, crash and heap dumps, temp files, unfinished downloads,
  browser/IDE/Unreal/package caches, installers and archives in Downloads,
  and more. The footer adds up how much of the listed space each hint covers.
- **Safe traversal.** Symlinks, junctions and other reparse points are not
  followed, so nothing is counted twice and the scan cannot loop.
- **Interruptible.** Press Ctrl+C during a scan to stop it and see the results
  found so far.
- **Show in Explorer.** Type a result's number to open its location in
  Explorer, Finder or your file manager.
- **CSV export.** Results, hints included, open correctly in Excel, even with
  non-ASCII paths (UTF-8 with BOM, `;` separator).
- **Cross-platform.** Built for Windows first; also works on Linux and macOS.

## Installation

**Prebuilt binaries:** download the archive for your OS from
[Releases](https://github.com/gsaranov/discscan/releases), unpack it and run
`discscan` (`discscan.exe` on Windows). On Windows you can also just
double-click the `.exe`.

**With Go 1.26+:**

```sh
go install github.com/gsaranov/discscan@latest
```

**From source:**

```sh
git clone https://github.com/gsaranov/discscan
cd discscan
go build -ldflags "-s -w" .
```

## Usage

### Interactive

Run `discscan` with no arguments, choose a mode, then a drive (or type any
folder path) and answer a couple of questions. Press Enter to accept the
default shown in brackets.

### Command line

```
discscan largest [flags] [path]   largest files
discscan recent  [flags] [path]   large files that appeared recently
discscan folders [flags] [path]   largest folders
discscan apps    [flags] [path]   programs, games and app data by total size
discscan drives                   list drives with free space
discscan version                  print version
```

`path` defaults to the current directory. You can put flags before or after
the path.

| Command   | Flag       | Default | Description                                         |
|-----------|------------|---------|-----------------------------------------------------|
| `largest` | `-n`       | `50`    | number of files to show                             |
|           | `-min`     | `0`     | ignore smaller files (`100MB`, `1.5GB`, …)          |
| `recent`  | `-min`     | `100MB` | ignore smaller files                                |
|           | `-within`  | `7d`    | look-back period (`24h`, `3d`, `2w`, `90m`)         |
|           | `-n`       | `100`   | maximum number of files to show                     |
| `folders` | `-n`       | `30`    | number of folders to show                           |
|           | `-depth`   | `0`     | only list folders up to N levels deep (`0` = any)   |
| `apps`    | `-n`       | `30`    | number of folders to show                           |
| all       | `-csv`     |         | also save results to a CSV file                     |
|           | `-workers` | 2×CPU   | parallel directory readers                          |
|           | `-q`       | off     | hide the progress line                              |
|           | `-v`       | off     | list read errors after the results                  |
|           | `-no-hints`| off     | hide cleanup hints                                  |

Sizes accept `B`, `KB`, `MB`, `GB` and `TB` (binary, 1 KB = 1024 B). A
number without a unit is treated as megabytes. Periods without a unit are
days.

### Examples

```sh
# Top 100 largest files on drive C:
discscan largest -n 100 C:\

# Files of 1 GB or more that appeared on D: in the last 3 days
discscan recent -min 1GB -within 3d D:\

# Biggest folders in your profile, two levels deep, saved to CSV
discscan folders -depth 2 -csv folders.csv %USERPROFILE%

# Which programs and games take the most space on C:
discscan apps -n 50 C:\

# Daily report from Task Scheduler / cron
discscan recent -q -min 500MB -within 24h -csv daily.csv C:\
```

### Tips

- Run the terminal **as Administrator** to include system folders. Without it,
  items that cannot be read are skipped and counted in the summary. Add `-v`
  to list them.
- Hints are advice, not commands: discscan never deletes anything. Check the
  item yourself before removing it, especially anything marked `careful`.
- Sizes are logical file sizes, the same as the "Size" column in Explorer.
  Sparse, compressed and cloud placeholder files may take less space on disk.
- Set `NO_COLOR=1` to turn off colored output.

## Project layout

```
main.go                 entry point
internal/
  cli/                  subcommands, interactive menu, Ctrl+C handling
  finder/               analysis modes: largest, recent, folders, apps
  hints/                cleanup advice rules for well-known files and folders
  scan/                 parallel directory walker
  top/                  generic bounded top-N heap
  report/               console tables and CSV export
  term/                 colors and progress line
  units/                size and duration parsing and formatting
  platform/             drives, "show in Explorer", console setup (per OS)
```

## Development

```sh
go test -race ./...
go vet ./...
```

Releases are built by [GoReleaser](https://goreleaser.com) when you push a tag:

```sh
git tag v1.0.0 && git push origin v1.0.0
```

## License

[MIT](LICENSE)
