# AGENTS.md

Guidance for coding agents working in this repository.

## Project

`quest` (QuesTUI) is a local, terminal-based TUI for learning and practising
programming through guided projects and programming problems. It aims to be a quick
interface to solve problems and to view, build, test and run those projects locally.

Provider support, in order of intent:

- **CodeCrafters** — course specs are MIT; tester repos mostly have **no
  license**. Current focus.
- **Advent of Code** — must behave as a polite HTTP client (no bulk scraping).
- **Exercism** — future.
- Further open-source guided projects.

The project is under active development. Do not assume an API exists just
because a type or constant does; much of `internal/` is scaffolding.

## Layout

```
cmd/quest/                intended entry point for the TUI (currently a stub)
cmd/test/                 scratch main, currently decodes a local course-definition.yml
internal/codecrafters/    CodeCrafters domain model
  definitions.go          CourseDefinition and nested types, mirrors course-definition.yml
  sources.go              pinned upstream sources: spec commits, tester releases, asset hashes
  README.md               upstream repo table with license status
temp/                     gitignored scratch clones of upstream courses (reference only)
```

Module: `github.com/aadhii-yz/quest`. Go 1.27.1. Only dependency is
`gopkg.in/yaml.v3`. There is no Makefile, no CI config and no test suite yet —
do not invent commands that do not exist.

`cmd/quest` is where the TUI belongs. If `go build ./...` fails with
"function main is undeclared in the main package", `cmd/quest/main.go` is a
stub, not a regression — do not delete it or add a dummy `main` to silence it.

## Commands

```sh
go build ./...
go vet ./...
gofmt -l .                # must print nothing before committing
go run ./cmd/test         # needs temp/build-your-own-grep/course-definition.yml present
```

No tests exist. If you add tests, they are plain `go test` packages; there is
no test harness to extend.

## Licensing and upstream safety

This is the highest-risk area of the project. The tester binaries are third
party code executed on a user's machine.

- **Never vendor, mirror, bundle or copy tester source into this repo.** Most
  tester repos have no license, meaning all rights reserved. Cloning or
  redistributing them would also appear to fall under this repo's MIT `LICENSE`,
  which would be a grant this project has no right to make.
- **`kafka-tester` is the only MIT-licensed tester.** Do not generalise from it.
- **Never fetch tester source at runtime.** The design is: download a single
  prebuilt release asset per platform, verify its SHA256, execute it. Do not
  add a `git clone` path.
- **Never resolve `"latest"` at runtime.** Releases are pinned in
  `sources.go`. `TesterRelease` is also required to construct asset filenames,
  so runtime resolution would break the hash guarantee.
- **Always verify SHA256 before extracting or executing** an asset, and fail
  closed on mismatch. A mismatch is not a warning.
- **Downloading is not permission.** `TesterLicense` exists to gate behaviour:
  do not download or run an `LicenseNone` tester without explicit, specific
  user consent that names the missing license.
- Downloaded assets must live in a user-visible, user-deletable cache directory
  (created `0700`), never inside the repo tree or a release artifact.
- Do not implement anything that decompiles a tester binary to extract test
  cases or expected outputs.

When adding a provider, document its licensing posture before writing code, and
extend `LicenseStatus` rather than encoding license state in booleans or free
text.

## Updating pinned sources

`sources.go` pins, per course: spec repo commit, tester release tag, the
checksums.txt URL, and every published asset URL + SHA256 keyed by `Arch*`
constants.

To bump a pin:

1. Resolve the new value from the GitHub API (`/commits?per_page=1` for spec
   repos, `/releases/latest` for testers).
2. Read the release's published `*_checksums.txt` and take digests from there.
3. Record the **asset set exactly as published**. Do not synthesise a platform
   that has no asset. Platform coverage genuinely differs per course; check the
   checksum file rather than assuming a matrix.
4. Re-verify by downloading every asset and confirming SHA256. Assert no
   release asset is unaccounted for besides `checksums.txt`.
5. Update the commit body with any changed platform coverage.

Known gaps, verified against checksums.txt — re-check rather than trusting this
list:

- No `darwin_arm64`: grep, claude-code, interpreter, bittorrent, git,
  dns-server, docker, sqlite.
- No `linux_386`: shell, redis, kafka, http-server, sqlite.
- `sqlite-tester` assets are ~365 MB each (embedded sqlite binaries). Any UI
  that starts a download must show size before beginning.

Use `Source.AssetFor` for lookups. Do not index `Assets` directly: a missing
platform must be reported to the user, not read as a zero-valued `Asset`.

## Code conventions

- YAML tags in `definitions.go` mirror upstream field names exactly. Treat them
  as a wire format: do not rename, and do not drop `omitempty` from
  `PrimaryExtensionSlug` — base stages omit that key entirely.
- Unknown YAML keys are intentionally ignored rather than rejected, so new
  upstream fields do not break parsing. `CourseDefinition` documents the known
  gaps (`concept_slugs`, `tester_source_code_url`, etc.).
- `CompletionPercentage` stays a `string`: upstream YAML is not guaranteed to
  be a plain integer.
- `Stage` order is significant; the slice index is the stage sequence.
- Prefer adding a field to `Source` over hardcoding upstream facts in UI or
  command code. `sources.go` is the single source of truth for pins and
  licenses.
- Standard library first. Justify any new dependency.

## Commit messages

Match existing history: lowercase, semicolon-separated clauses, no
Conventional Commits prefix.

```
pin spec commits; pin tester releases with per-arch assets and sha256;
```

Body: short bullets of what changed, plus how the change was verified when the
change touches pins, hashes or licensing facts.

End with the trailer:

```
Co-Authored-By: DeepSeek-V4.1-Flash via Pi Agent
```

## Do not

- Treat `cmd/test/` as a stable entry point; it is a scratch harness.
- "Fix" a `cmd/quest` build error by adding a placeholder `main` or by
  deleting the package.
- Add network calls to `definitions.go` or `sources.go`; both are static data
  and pure types.
