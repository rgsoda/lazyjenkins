# Installing lazyjenkins on Linux

`lazyjenkins` is a lazygit-style terminal UI for Jenkins. Under the hood it
drives the [`jk`](https://github.com/avivsinai/jenkins-cli) CLI — the release
binaries ship with `jk` embedded, so you normally don't install anything else.

Supported: Linux `x86_64` (amd64) and `arm64`.

---

## Method A — Prebuilt binary (recommended)

No toolchain needed. The tarball contains a single static binary with `jk`
bundled in.

```bash
# Detect arch and grab the latest release
ARCH=$(uname -m); case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "unsupported arch: $ARCH"; exit 1 ;;
esac

curl -sSfL "https://github.com/rgsoda/lazyjenkins/releases/latest/download/lazyjenkins_linux_${ARCH}.tar.gz" \
  | tar -xz lazyjenkins

# System-wide:
sudo install -m 0755 lazyjenkins /usr/local/bin/lazyjenkins
# …or just for you (make sure ~/.local/bin is on PATH):
# install -m 0755 lazyjenkins ~/.local/bin/lazyjenkins

rm lazyjenkins
lazyjenkins --help
```

### Verify the download (optional)

```bash
curl -sSfLO https://github.com/rgsoda/lazyjenkins/releases/latest/download/checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

### Updating

Re-run the same commands — it overwrites the old binary.

---

## Method B — Build from source (Makefile)

Requires **Go 1.27+** (`go version`) and, for the bundled build, `curl` + `tar`.

```bash
git clone https://github.com/rgsoda/lazyjenkins.git
cd lazyjenkins

make install          # builds with jk embedded, installs to /usr/local/bin (asks for sudo)
# or, no sudo:
make install-user     # installs to ~/.local/bin
```

Other targets (`make help` lists them all):

| Target               | What it does                                                        |
| -------------------- | ------------------------------------------------------------------- |
| `make build`         | Plain binary. Needs `jk` on your `PATH` at runtime.                 |
| `make build-bundled` | Binary with `jk` embedded (downloads the pinned `jk` first).       |
| `make install`       | `build-bundled` + install into `$(PREFIX)/bin` (default `/usr/local`). |
| `make install-user`  | `build-bundled` + install into `~/.local/bin`.                      |
| `make uninstall`     | Remove the installed binary.                                        |
| `make run`           | Build and launch.                                                   |
| `make clean`         | Remove build artifacts.                                             |

Custom location:

```bash
make install PREFIX=/opt/lazyjenkins          # -> /opt/lazyjenkins/bin/lazyjenkins
make install DESTDIR=/tmp/pkg                 # staged install for packaging
```

If you'd rather manage `jk` yourself, use `make build` and put `jk` on your
`PATH`, or point at it explicitly: `lazyjenkins --jk-bin /path/to/jk`.

---

## Method C — Homebrew (Linuxbrew)

```bash
brew install rgsoda/lazyjenkins/lazyjenkins
```

---

## Method D — Nix

If you use Nix, `lazyjenkins` is fully packaged and reproducible via its flake.

### Run directly
Execute `lazyjenkins` instantly without installing it:
```bash
nix run github:rgsoda/lazyjenkins
```

Or run the underlying `jk` CLI dependency directly:
```bash
nix run github:rgsoda/lazyjenkins#jk -- --help
```

### Install profile-wide
Install `lazyjenkins` into your Nix user profile:
```bash
nix profile install github:rgsoda/lazyjenkins
```

### Development environment
Enter a completely pre-configured developer shell containing Go, Gnumake, `golangci-lint`, and the `jk` dependency on your path:
```bash
nix develop
```

---

## First run

`lazyjenkins` forwards any subcommand straight to the bundled `jk`, so you can
configure everything without a separate `jk` install:

```bash
# Log in to a Jenkins instance (token from Jenkins > User > Security > API Token)
lazyjenkins auth login https://jenkins.example.com --token "$JENKINS_TOKEN"

lazyjenkins auth status      # check it worked
lazyjenkins                  # launch the TUI
```

Multiple Jenkins instances:

```bash
lazyjenkins auth login https://ci.other.com --token ... --context other
lazyjenkins context ls
lazyjenkins --context other          # or set JK_CONTEXT=other
```

With 2+ contexts configured and none pinned, `lazyjenkins` shows a picker on
startup.

### Credential storage

`jk` keeps its config under `~/.config/jk/` and stores the **API token in your OS
keyring** (Secret Service) rather than in a plaintext file. On a headless box or
a minimal WM you need a keyring daemon running, e.g.:

- `gnome-keyring-daemon` (GNOME), or
- `kwalletd` (KDE), or
- for scripts/CI, export the token as an env var instead — see
  `lazyjenkins auth --help`.

---

## Key bindings (in the TUI)

| Key            | Action                                            |
| -------------- | ------------------------------------------------- |
| `1` `2` `3`    | Jump to Jobs / Runs / Main panel (`4` = Debug)    |
| `tab`          | Cycle panels                                      |
| `↑`/`k` `↓`/`j`| Move selection                                    |
| `enter`        | Open job / folder, or view a run's log            |
| `esc`          | Back out of a folder / close the log or form      |
| `/`            | Filter the current list (or search within a log)  |
| `n` / `N`      | Next / previous search match in a log             |
| `w`            | Toggle log line wrapping                          |
| `s`            | Start a run (opens the parameters form)           |
| `R`            | Re-run the selected build                         |
| `c`            | Cancel the selected / running build               |
| `y`            | Copy the selected build's Jenkins URL             |
| `+` / `-`      | Resize the left column                            |
| `r`            | Force refresh                                     |
| `q` / `ctrl+c` | Quit                                              |

Run with `lazyjenkins --debug` to get a `[4] Debug` panel logging every `jk`
call (args, timing, errors) — useful when a build isn't showing up.

---

## Troubleshooting

| Symptom                                   | Fix                                                                            |
| ----------------------------------------- | ----------------------------------------------------------------------------- |
| `lazyjenkins: exec: "jk": not found`      | You built with `make build`. Use `make build-bundled` / a release, or install `jk`. |
| `command not found: lazyjenkins`          | `~/.local/bin` isn't on `PATH`. Add `export PATH="$HOME/.local/bin:$PATH"` to your shell rc. |
| Token not saved / prompts every time      | No keyring daemon running — see *Credential storage* above.                    |
| `go: ... requires go >= 1.27`             | Update Go, or use Method A (prebuilt).                                         |
| TLS / self-signed cert errors             | Configure it on the `jk` side: `lazyjenkins auth login --help`.               |
| Nothing loads, no error                   | `lazyjenkins auth status`; re-run `lazyjenkins auth login` if the context is wrong. |
