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
- Content bundles pipeline exists in `quest-source/` and builds all 12 CodeCrafters
  courses; no release has been published and quest does not consume it yet.

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

- [ ] Add quest CI: build, vet, gofmt, `go test ./... -race` on linux, macOS and
      Windows.
- [ ] Add quest release workflow on `v*` tags: matrix binaries, `checksums.txt`,
      `--version` from `debug.ReadBuildInfo()`.
- [ ] Extend `internal/config` with tester backend selection
      (`tester = "direct" | "codecrafters-cli"`, `codecrafters_bin`), file +
      env + flag precedence.
- [ ] Decide artifact plumbing for the direct backend: add
      `provider.Artifact{URL, SHA256, Filename}` (or similar) to
      `provider.TesterPlan`, filled by the CodeCrafters adapter from pins, so
      `cache`/`runner` never import a provider implementation.
- [ ] Spike (no code): establish how a pinned tester binary is invoked — args,
      env, working directory, git requirements. Try `--help` in `temp/`.
      Record findings in `internal/runner/doc.go`.

## Phase 1 — real test loop

- [ ] `runner.Wrapper`: run the user's `codecrafters` CLI, stream output as
      `runner.Event`s, cancellable. Tests use a fake CLI on `PATH`.
- [ ] `cache.Fetch`: download one pinned asset with progress and total size
      (sqlite assets are ~365 MB), temp file, verify, atomic rename, fail closed.
- [ ] `cache.Extract`: safe tar.gz extraction into a 0700 cache with traversal,
      link, duplicate and size protections.
- [ ] `licensing` consent store: persisted in the cache dir, keyed by course +
      release + digest; UI modal names the missing license; no approval, no
      fetch. Approval never carries to a new release/digest.
- [ ] `runner.Direct`: exec the verified binary with `exec.CommandContext`,
      sanitized env, streamed lines and exit status. Tests use a fake tester.
- [ ] UI: show the active backend, download size before starting, progress, and
      consent prompt. Replace `Simulated` or keep it as an explicit dev mode.

Done when: grep/go runs a real stage from an empty cache, second run is offline,
and a tampered asset or missing consent blocks execution.

## Phase 2 — workspace

- [ ] `internal/workspace`: project directory per course (default under the
      user's home, override in config), starter template selection from content
      bundles, language toolchain detection.
- [ ] Build/run the user's solution (`b` / `R`) as a separate runner, streamed
      into the log pane.
- [ ] Editor handoff (`e`) via `$EDITOR`.
- [ ] `internal/state`: last course/language/stage and completed stages in the
      cache dir, atomic JSON writes.

## Phase 3 — content integration

- [ ] Publish `content-0.1.0` from `quest-source/` (all 12 bundles + index +
      checksums; `sha256sum -c` verified).
- [ ] Pin `ContentRelease`, `ContentRepo` and `ContentIndexSHA256` in the
      CodeCrafters adapter.
- [ ] Fetch the index through `internal/cache`, verify the digest, cache it;
      render the catalog from the index without downloading bundles.
- [ ] Fetch and extract a course bundle on demand, validate manifest
      `format`/provider/course/`spec.commit`/license against quest pins, then
      map it to `provider.Course`.
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
   or keep direct execution primary and treat the wrapper as an option?
