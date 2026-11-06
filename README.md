# QuesTUI (`quest`)

Quest is a local TUI to learn and master programming. This project's goal is to provide an quick interface to solve problems, view, build, test and run guided projects.

> [!CAUTION]
> Currently under development

## Install

Requires Go 1.27 or newer.

```sh
go install github.com/aadhii-yz/quest/cmd/quest@latest
```

Prebuilt binaries for Linux, macOS and Windows are on the
[releases](https://github.com/aadhii-yz/quest/releases) page. Each release
publishes `checksums.txt`; verify a download with
`sha256sum -c checksums.txt`.

```sh
quest --version
```

Inside a starter directory (one containing `.quest/state.json`), quest opens
that course at the saved stage instead of the catalog. On a course screen,
press `c` to create a starter directory: it copies the verified starter
templates, writes the marker and adds `.quest/` to the project's `.gitignore`.

Course content is loaded from the pinned `quest-source` release and verified
against its index and bundle digests. A course bundle is fetched and verified
the first time you open that course; later runs use the cache. `QUEST_DEFS_DIR` overrides that with a
local directory of `course-definition.yml` files for development, and
`QUEST_CACHE_DIR` moves the cache; see [AGENTS.md](AGENTS.md).

Initially I am planning to support:

- [CodeCrafters](https://codecrafters.io/): Their course specifications are under MIT License. The tester repos have no license, so thinking of wrapping their cli from the TUI.
- [Advent of Code](https://adventofcode.com/): Respect to [the creator's concern](https://www.reddit.com/r/adventofcode/comments/3v64sb/aoc_is_fragile_please_be_gentle/), will act as proper HTTP client.
- [Exercism](https://exercism.org/): Future
- more open source guided projects, programming problems and courses planned

> [!DISCLAIMER]
> This is not an official client of any of the resource providers used. Every resource is used under fair use.
