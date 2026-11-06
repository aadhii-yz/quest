# Roadmap

Working plan for quest. Update this file when a decision lands, so a new session
can pick up without re-deriving context. Architecture rules live in
[AGENTS.md](AGENTS.md); the content pipeline lives in the nested, gitignored
repo at `quest-source/` (see `quest-source/AGENTS.md` and `quest-source/docs/`).

## Current state

- Two-screen TUI (catalog, course) with toggleable stage and test panes.
- Providers: CodeCrafters (local `course-definition.yml` discovery), Advent of
  Code (pinned years, placeholder stages), Exercism / open-source placeholders.
- Tester execution is a labelled **simulation** (`internal/runner.Simulated`).
- `internal/cache` has a 0700 store and fail-closed SHA256 verification, but no
  download or extraction yet.
- `internal/licensing` has the `LicenseNone` policy, but no persisted consent.
- Content bundles pipeline exists in `quest-source/` and `content-0.1.0` is
  published with all 12 CodeCrafters courses.
- `internal/content` verifies the pinned index, fetches bundles on demand and
  checks the manifest against quest's spec pins.
- Opening a course in the TUI loads its bundle on a `tea.Cmd`, shows a loading
  state and refreshes the course; `QUEST_DEFS_DIR` still wins as a dev
  override. Catalog metadata still comes from `sources.go`.
- Projects exist: starting quest in a directory with `.quest/state.json` opens
  its course at the saved stage, and `c` creates a starter directory from the
  verified bundle.

## Decisions made

- **Layering:** `cmd/quest` wires; `internal/ui` renders; `internal/provider`
  describes; `internal/runner` executes; `internal/cache` verifies; 
  `internal/licensing` gates. No UI imports in domain packages.
- **Tester backends will be swappable:** a `runner.Backend` interface with a
  direct (pinned asset) implementation and a wrapper (user-installed
  `codecrafters` CLI) implementation. Explicit selection only; no silent
  fallback. Consent applies to the direct backend.
- **Content is prebuilt, pinned and verified:** per-course bundles plus an
  index in `quest-source/`, consumed via the same verified-download path as testers.
  The app validates the bundle manifest's `spec.commit` against its own pins.
- **Compatibility:** content format version and quest pins move together; a
  bundle from a different spec commit is rejected.

## Phase 0 — guardrails (next)

- [x] Add quest CI: build, vet, gofmt, `go test ./... -race` on linux, macOS and
      Windows.
- [x] Add quest release workflow on `v*` tags: matrix binaries, `checksums.txt`,
      `--version` from `debug.ReadBuildInfo()`.
- [ ] Extend `internal/config` with tester backend selection
      (`tester = "direct" | "codecrafters-cli"`, `codecrafters_bin`), file +
      env + flag precedence.
- [x] Decide artifact plumbing for the direct backend: add
      `provider.Artifact{URL, SHA256, Filename, Size}` to `provider.TesterPlan`,
      filled by the CodeCrafters adapter from pins, so `cache`/`runner` never
      import a provider implementation.
- [x] Spike (no code): establish how a pinned tester binary is invoked.
      Findings are in `internal/runner/doc.go`: testers read
      `CODECRAFTERS_SUBMISSION_DIR` and `CODECRAFTERS_TEST_CASES_JSON` (stage
      metadata, not test logic), and expect the course entrypoint script plus
      `codecrafters.yml` from the starter templates. Exact argv, working
      directory and tty needs are unverified until an unlicensed tester is run
      with consent. The wrapper needs a CodeCrafters-cloned workspace; CLI v54
      has no login command and refuses other remotes.

## Phase 1 — real test loop

- [ ] `runner.Wrapper`: run the user's `codecrafters` CLI, stream output as
      `runner.Event`s, cancellable. Tests use a fake CLI on `PATH`.
- [x] `cache.Fetch`: download one pinned asset with progress and total size
      (sqlite assets are ~365 MB), temp file, verify, atomic rename, fail closed.
      A verified cached file is reused without network access.
- [x] `cache.Extract`: safe tar.gz extraction into a 0700 cache with traversal,
      link, duplicate and size protections; extracts to a temp directory and
      renames only on success.
- [ ] `licensing` consent store: persisted in the cache dir, keyed by course +
      release + digest; UI modal names the missing license; no approval, no
      fetch. Approval never carries to a new release/digest.
- [ ] `runner.Direct`: exec the verified binary with `exec.CommandContext`,
      sanitized env, streamed lines and exit status. Build
      `CODECRAFTERS_TEST_CASES_JSON` from the course definition and confirm the
      log-prefix numbering for extension stages. Tests use a fake tester.
- [ ] UI: show the active backend, download size before starting, progress, and
      consent prompt. Replace `Simulated` or keep it as an explicit dev mode.

Done when: grep/go runs a real stage from an empty cache, second run is offline,
and a tampered asset or missing consent blocks execution.

## Phase 2 — workspace

- [x] `internal/workspace`: detect `.quest/state.json` by walking up from the
      working directory, save it atomically at 0600, and create starter
      directories from verified bundles (`all/code`, then `<language>/code`)
      without overwriting a non-empty destination.
- [x] TUI create flow: `c` on the course screen prompts for a directory, copies
      the templates, writes the marker, appends `.quest/` to the project's
      `.gitignore`, and reports the result in the test log.
- [x] Start inside a project on its saved course, language and stage; moving
      the stage or switching language persists the marker. A broken marker is
      reported and the catalog still opens.
- [ ] Build/run the user's solution (`b` / `R`) as a separate runner, streamed
      into the log pane.
- [ ] Language toolchain detection from the starter `config.yml`
      (`required_executable`, `user_editable_file`).
- [ ] Editor handoff (`e`) via `$EDITOR`.
- [ ] Global state beyond the per-project marker (last catalog position) only
      if it proves necessary.

## Phase 3 — content integration

First release is live and verified:

```
release:  https://github.com/aadhii-yz/quest-source/releases/tag/content-0.1.0
repo:     aadhii-yz/quest-source
index:    sha256 c0f520f9053980eda7759cf43de20618a2f5ec50f9f2353ce45e3af744975dce
assets:   12 course bundles + index.json + checksums.txt (all checksums pass)
```

- [x] Publish `content-0.1.0` from `quest-source/` (all 12 bundles + index +
      checksums; `sha256sum -c` verified).
- [x] Pin `ContentRelease` (`content-0.1.0`), `ContentRepo`
      (`aadhii-yz/quest-source`) and `ContentIndexSHA256` (`c0f520f9…`) in
      `internal/content`, and add the verified loader: index digest, bundle
      digest and size, safe extraction, manifest format/provider/course, spec
      commit and license against quest pins, and every manifest file digest.
- [x] Wire the loader into the app: inject `ui.ContentLoader` from `cmd/quest`,
      load a bundle on course open via `tea.Cmd`, show a loading state and
      replace the providers, course and catalog selection afterwards.
- [ ] Render catalog names and summaries from the cached index instead of
      `sources.go`; keep the pin fallback when the index is unavailable.
- [ ] Map a verified bundle to `provider.Course` (definition + starter
      templates) and keep `sources.go` as the fallback when no bundle exists.
- [ ] Precedence: `QUEST_DEFS_DIR` (dev) > verified cached bundle > index-only.
- [ ] Resolve async loading: `provider.Provider` is synchronous today. Either
      add `Load(ctx)` called from a `tea.Cmd` with loading state, or load cached
      index synchronously and refresh in the background. Decide before coding.
- [ ] Decide whether stage text comes from `course-definition.yml`
      `marketing_md` or the bundle's `stage_descriptions/<slug>.md`.

## Phase 4 — Advent of Code

- [ ] Real provider with a rate-limited HTTP client, explicit User-Agent,
      per-day caching, no bulk scraping.
- [ ] Input files/session handling documented (AoC input requires a login).
- [ ] Never redistribute AoC puzzle text; no AoC content in bundles.

## Phase 5 — hardening and polish

- [ ] Fuzz tests: YAML definitions, bundle extraction, cache name validation.
- [ ] Re-verify cached digests immediately before execution (`cache.VerifyFile`).
- [ ] Real progress bars, reusable consent modal, stage completion markers,
      log search/copy, mouse support.
- [ ] Update AGENTS.md and this file with every decision.

## Open decisions

1. Backend default and discovery UX: default `direct` with a hint when the CLI
   is present, or ask on first run?
2. Consent key granularity and where the record lives (cache dir vs XDG config).
3. Content consumption model: extend `provider.Provider` with async loading, or
   add a separate content store the adapter consults synchronously?
4. Whether content releases are pinned per app release only, or also
   user-updatable via a digest-pinned config channel.
5. Starter template subsetting: ship every language from the bundle (current) or
   fetch only the selected language? Bundles are ~20-200 KB today, so ship all.
6. Long-term: wrap the CodeCrafters CLI as the primary path (auth, submissions)
   or keep direct execution primary and treat the wrapper as an option? The
   direct path depends on reproducing CodeCrafters' stage metadata and
   entrypoint expectations; the wrapper depends on a CI-cloned workspace.
7. Does an extension stage number reset to 101 per extension, or run
   cumulatively? Needed for `tester_log_prefix`; verify in the first consented
   tester run instead of guessing.
