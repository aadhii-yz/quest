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
because a type or constant does; some packages are scaffolding for work that is
not wired up yet. Current phases and open decisions live in
[ROADMAP.md](ROADMAP.md).

## Layout

```
cmd/quest/main.go             thin entry point: config, provider wiring, tea.NewProgram

internal/config/              user paths: cache dir, local definitions dir
internal/provider/            contracts every problem source implements
  provider.go                 Provider, Course, Stage, TesterPlan, LicenseStatus
  aoc/                        Advent of Code (static years for now)
  codecrafters/
    definitions.go            CourseDefinition and nested types, mirrors course-definition.yml
    sources.go                pinned upstream sources: spec commits, tester releases, asset hashes
    adapter.go                adapts the pinned sources + definitions to provider.Course
    README.md                 upstream repo table with license status
internal/cache/               user cache directory, asset storage, SHA256 verification
internal/content/             verified quest-source release: index, bundles, manifest checks
internal/licensing/           license policy: which testers need explicit consent
internal/runner/              tester execution contract; simulated runner for now
internal/ui/                  Bubble Tea / Lip Gloss / Glamour code
  app.go                      root model, routing, layout, chrome
  catalog.go                  catalog screen: resources, courses, language
  course.go                   course screen: stages, description, test output
  theme.go                    palette, glamour renderer, help/list styles
  panel.go                    panel geometry and text fitting
  rows.go                     shared list row delegate
  keys.go                     key maps and help bindings
quest-source/                 separate git repo (gitignored here): per-course content bundles
temp/                         gitignored scratch clones used only with QUEST_DEFS_DIR
```

Module: `github.com/aadhii-yz/quest`. Go 1.27.1.

Dependencies: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`,
`charm.land/lipgloss/v2`, `charm.land/glamour/v2`, `gopkg.in/yaml.v3`. Standard
library first; justify any addition.

## Architecture rules

Dependencies point one way:

```
cmd/quest -> internal/ui -> internal/provider, internal/runner, internal/cache, internal/licensing
cmd/quest -> internal/content
internal/content -> internal/cache, internal/provider
internal/ui -> internal/config            (paths only)
internal/provider/codecrafters -> internal/provider, YAML wire types and pins
internal/cache, internal/runner, internal/licensing -> internal/provider
```

- `internal/provider` and its adapters must not import Bubble Tea, Lip Gloss,
  Glamour or any other UI package. They describe what exists; they never
  render, prompt or fetch tester binaries.
- `internal/ui` must not import `internal/provider/codecrafters` or any
  provider implementation. It works only with `provider.Provider`,
  `provider.Course`, `provider.Stage` and `provider.TesterPlan`, which keeps
  new providers additive.
- Providers expose availability and license posture through `TesterPlan`. They
  must not download or execute testers; that belongs to `internal/cache`,
  `internal/licensing` and `internal/runner`.
- Execution output reaches the UI as `runner.Event` values. Do not fabricate
  tester output, verification steps or pass/fail results in UI code.
- Content fetching lives in `internal/content` and is injected into the UI as a
  `ui.ContentLoader` built in `cmd/quest`. The UI must not import
  `internal/content` or any provider implementation, and must not fetch by
  itself.
- Read environment variables and user paths in `internal/config` or in
  `cmd/quest`, not inside domain packages. Inject paths (for example the local
  definitions directory) rather than calling `os.Getenv` deep in an adapter.

## Content repository (track B)

`quest-source/` is a **separate git repository**, ignored by this one. It pins the
spec commits of the MIT course repositories and publishes immutable per-course
bundles (course-definition.yml, LICENSE, stage_descriptions, starter_templates,
manifest.json), plus an index and checksums. Its rules live in
`quest-source/AGENTS.md`; the format and consumer contract live in
`quest-source/docs/bundle-format.md` and `quest-source/docs/quest-integration.md`.

- Never commit the nested repository or its `dist/` output to quest, and never
  copy bundle files into this tree.
- When a spec commit is bumped in `internal/provider/codecrafters/sources.go`,
  the same commit must be pinned in `quest-source/courses.json` and a new content
  release cut. Do not let the two drift.
- Content is consumed only through `internal/cache` verification: pin a release
  tag and the `index.json` digest, verify the bundle digest and size, extract
  safely, then require the manifest `spec.commit` to equal this repo's pin.
  Never resolve a content "latest" release.
- Content bundles are data. Never execute anything from a bundle, and never let
  a bundle carry tester source, tester binaries or executable assets.
- `internal/content` is the only consumer: it pins the release and index
  digest, fetches through `internal/cache`, verifies the bundle and every
  manifest file digest, and rejects a manifest whose `spec.commit` differs from
  `sources.go`. Do not add a second downloader or a "latest" path.
- `temp/` is scratch space, never a default content source. Only an explicit
  `QUEST_DEFS_DIR` may point at local definitions for development; do not
  reintroduce a default that reads the working directory.

## Commands

```sh
go build ./...
go vet ./...
gofmt -l .                # must print nothing before committing
go test ./...
go run ./cmd/quest        # uses verified quest-source content from the cache
```

The content repository has its own module and commands; see
`quest-source/AGENTS.md` (`cd quest-source && go test ./...`).

`QUEST_DEFS_DIR` overrides the local definitions directory; `QUEST_CACHE_DIR`
overrides the cache directory. There is no Makefile and no CI config; do not
invent commands that do not exist.

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
  `internal/provider/codecrafters/sources.go`. `TesterRelease` is also required
  to construct asset filenames, so runtime resolution would break the hash
  guarantee.
- **Always verify SHA256 before extracting or executing** an asset, and fail
  closed on mismatch. A mismatch is not a warning. Verification belongs in
  `internal/cache`; never add a code path that extracts or executes an asset
  without it.
- **Downloading is not permission.** `provider.LicenseStatus` exists to gate
  behaviour: do not download or run an `LicenseNone` tester without explicit,
  specific user consent that names the missing license. Consent policy lives
  in `internal/licensing`, not in the UI.
- Downloaded assets must live in a user-visible, user-deletable cache directory
  (created `0700`), never inside the repo tree or a release artifact.
- Do not implement anything that decompiles a tester binary to extract test
  cases or expected outputs.

When adding a provider, document its licensing posture before writing code, and
extend `LicenseStatus` rather than encoding license state in booleans or free
text.

## Updating pinned sources

`internal/provider/codecrafters/sources.go` pins, per course: spec repo commit,
tester release tag, the checksums.txt URL, and every published asset URL +
SHA256 keyed by `Arch*` constants.

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

Domain and wire format:

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
- Provider interface methods stay small and data-only (`ID`, `Name`, `Note`,
  `Planned`, `Courses`). Adding network fetching later means returning data
  through a command/service, not adding `Run`/`Download` to a provider.

UI:

- One `internal/ui` package. Split files by screen (`catalog.go`, `course.go`),
  not into a package per pane; panes share selection and focus state. Add an
  `internal/ui/components` subpackage only when a widget is independent of the
  theme and genuinely reused.
- Palette, borders, spacing and Markdown rendering live in `theme.go`. Panel
  geometry (`panel`, `panelInner`, `fit`, `bar`) lives in `panel.go` and must
  stay in one place. In Lip Gloss v2, `Width`/`Height` include border and
  padding — remember this when sizing panes.
- List rows implement the shared `rowItem`/`rowDelegate` in `rows.go`; set the
  list's width to the panel's inner content width or rows will wrap.
- Key bindings and help text live in `keys.go`; every screen exposes a
  `help.KeyMap`. Do not hardcode footer hint strings.
- Components are updated in `Update`; the root model forwards unrecognised
  messages to the focused component (filter matches, cursor blinks, runner
  events). Swallowing component messages breaks Bubble Tea components.

Runner and cache:

- `internal/runner` emits plain-text events in order, ending with a done event.
  The UI styles them; it must not invent them. Real execution must be
  cancellable with `context.Context`.
- `internal/cache` owns the `0700` directory, extraction and SHA256 checks. It
  must offer no "skip verification" option. Show asset size before starting a
  download of a large asset.

Tests:

- Plain `go test` packages, no harness. Keep unit tests free of network access
  and of real tester execution.
- Cover provider adapters (definition discovery, stage mapping, platform
  availability), cache verification (mismatch fails closed), licensing policy
  and runner event ordering.
- Keep at least one render-invariant test in `internal/ui`: fixed window sizes,
  exact frame height, no line wider than the terminal, no panic at small
  sizes. Use fake providers; do not read `temp/`.

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

- Add UI imports to `internal/provider`, `internal/cache`, `internal/runner` or
  `internal/licensing`.
- Import a provider implementation from `internal/ui`.
- Put license, SHA256 or download policy in the UI, or skip SHA256 verification
  anywhere.
- Add a placeholder `main` or delete `cmd/quest` to silence a build error.
- Add network calls to `definitions.go` or `sources.go`; both are static data
  and pure types.
- Reintroduce `cmd/test/`; put definition parsing tests next to the adapter.
- Commit, vendor or copy the `quest-source/` repository or its release output into
  this tree.
- Fetch content bundles without verifying the pinned index and bundle digests,
  or trust a bundle whose manifest `spec.commit` differs from `sources.go`.
