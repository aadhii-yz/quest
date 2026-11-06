// Package runner executes testers and streams their output.
//
// The contract is deliberately small: a Request describes what to run and a
// channel of Events carries plain output lines followed by a done event.
// Implementations must stop producing events and close the channel when the
// context is cancelled; output is never fabricated.
//
// # Backends
//
// Two execution paths are planned. Neither is implemented yet; Simulated is
// the only Runner today and labels every line so the UI cannot imply real
// verification.
//
//   - Wrapper: run the user's installed codecrafters CLI in a workspace that
//     was cloned from CodeCrafters. The CLI owns authentication and the test
//     execution model; quest only sets the working directory, streams stdout
//     and stderr, and reports the exit status.
//   - Direct: fetch the pinned tester asset through internal/cache, verify its
//     SHA256, extract it, and execute it locally with the environment below.
//     This is gated by internal/licensing consent.
//
// # Tester invocation
//
// Findings from the public tester READMEs and the installed codecrafters CLI
// (v54), gathered without downloading or executing a tester:
//
//   - A tester is one prebuilt binary published as a per-platform tar.gz
//     asset. provider.TesterPlan.Artifact pins its URL, digest and size.
//
//   - The binary is configured through environment variables, at least:
//
//     CODECRAFTERS_SUBMISSION_DIR   root of the user's submission
//     CODECRAFTERS_TEST_CASES_JSON  JSON array of stage descriptors
//
//   - The test-cases JSON holds stage metadata, not test logic:
//
//     [{"slug":"cq2","tester_log_prefix":"stage-1","title":"Stage #1: ..."}]
//
//     The test logic lives in the binary. Slugs and titles exist in the course
//     definition, so quest can derive them. The log prefix uses CodeCrafters'
//     numbering, where extension stages start at 101; provider.Stage.Number is
//     a flattened sequence, so that mapping is still unverified.
//
//   - The tester expects the course's entrypoint script (for example
//     your_grep.sh, your_shell.sh or spawn_redis_server.sh) and a
//     codecrafters.yml in the submission directory. Both come from starter
//     templates, so preparing the workspace (internal/workspace) must precede
//     direct execution.
//
//   - The READMEs document no arguments; assume none until verified.
//
//   - Tester release notes mention pipe/pty handling. Start with stdout and
//     stderr pipes and keep a pty as a fallback if output is truncated.
//
// Still unknown: exact argv, working directory, any additional environment
// variables, and whether the process needs a tty. Resolving these requires
// downloading and running an unlicensed tester, which needs explicit consent
// under the licensing rules. Until then Direct stays unimplemented.
//
// # Wrapper notes
//
// codecrafters test [--previous] runs in a directory whose git remote is a
// CodeCrafters repository; v54 lists no login command and refuses to run
// elsewhere. The CLI resolves authentication itself, and quest must never
// manage tokens or clone repositories behind the user's back. The wrapper
// therefore needs the workspace directory on the Request once
// internal/workspace exists.
package runner
