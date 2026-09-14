# lazyjenkins

A lazygit-style terminal UI for Jenkins. It wraps the
[`jk`](https://github.com/avivsinai/jenkins-cli) CLI: browse jobs and folders,
watch runs, stream console logs, start/re-run/cancel builds, and fill in build
parameters — all without leaving the terminal.

## Features

- **Jobs panel** — navigate folders and multibranch pipelines, live status balls
- **Runs panel** — recent + queued builds, with parameters
- **Log view** — full or follow-mode streaming, word-wrap toggle, in-log search
- **Actions** — start a run (parameter form), re-run, cancel, copy the build URL
- **Multi-instance** — named `jk` contexts with a startup picker
- `--debug` panel logging every `jk` invocation (args, timing, errors)

The release binaries bundle `jk`, so there's nothing else to install.

## Installation

See **[INSTALL.md](INSTALL.md)** for Linux instructions (prebuilt binary,
build-from-source via the `Makefile`, Nix, and Homebrew), first-run setup, key
bindings, and troubleshooting.

Quick start (standard):

```bash
ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; esac
curl -sSfL "https://github.com/rgsoda/lazyjenkins/releases/latest/download/lazyjenkins_linux_${ARCH}.tar.gz" | tar -xz lazyjenkins
sudo install -m 0755 lazyjenkins /usr/local/bin/

lazyjenkins auth login https://jenkins.example.com --token "$JENKINS_TOKEN"
lazyjenkins
```

Quick start (Nix):

```bash
# Run lazyjenkins directly without installing
nix run github:rgsoda/lazyjenkins

# Or run the underlying jk CLI dependency directly
nix run github:rgsoda/lazyjenkins#jk -- --help
```

## Key bindings

| Key             | Action                                         |
| --------------- | ---------------------------------------------- |
| `1` `2` `3`     | Jump to Jobs / Runs / Main panel (`4` = Debug) |
| `tab`           | Cycle panels                                   |
| `↑`/`k` `↓`/`j` | Move selection                                 |
| `enter`         | Open job / folder, or view a run's log         |
| `esc`           | Back out of a folder / close the log or form   |
| `/`             | Filter list, or search within a log            |
| `n` / `N`       | Next / previous log search match               |
| `w`             | Toggle log line wrapping                       |
| `s` `R` `c`     | Start / re-run / cancel a build                |
| `y`             | Copy the selected build's Jenkins URL          |
| `+` / `-`       | Resize the left column                         |
| `r`             | Force refresh                                  |
| `q` / `ctrl+c`  | Quit                                           |

## Building

Requires Go 1.27+.

```bash
make build          # plain binary, expects `jk` on PATH
make build-bundled  # binary with `jk` embedded
make run            # build and launch
```

`make help` lists all targets.

### Nix

If you use Nix, you can enter a fully loaded development shell containing Go, Gnumake, `golangci-lint`, and `jk`:

```bash
nix develop
```

## License

MIT — see [LICENSE](LICENSE). Bundled third-party code is listed in
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).
