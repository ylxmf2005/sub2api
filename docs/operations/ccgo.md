# ccgo Reverse Workstation Operations

`ccgo` runs Claude Code on the platform while keeping the user's project files and developer commands on the user's machine.

The product invariant is strict: the canonical workspace is the user's local project root. The server workspace root is only a stable projection directory so Claude Code can bind its own project context and session history to a concrete directory.

## User Commands

```bash
ccgo login
ccgo login <token>
ccgo --server http://localhost:8080 login <token>
ccgo doctor [local_path]
ccgo .
ccgo status
ccgo stop
```

- `ccgo login` starts the device-code login flow, opens or prints the approval URL, polls until approval, and stores the issued platform token in the user's OS config directory.
- `ccgo login <token>` stores the platform token in the user's OS config directory.
- `ccgo --server <url> login <token>` or `CCGO_SERVER=<url> ccgo login <token>` stores a token for a non-default platform URL, which is the easiest local-development path.
- `ccgo doctor [local_path]` verifies the stored login config and, when a path is provided, runs the same local Bash preflight used by `ccgo <local_path>` without starting an agent or server runner.
- `ccgo <local_path>` resolves the canonical local path, creates or reuses the stable workspace mapping, starts the local agent, asks the server to start Claude Code, and records the last workspace id for `status` and `stop`.
- `ccgo status [workspace_id]` reports the workspace mapping, local-agent connection, projection/runner health, and latest workstation run.
- `ccgo stop [workspace_id]` stops the server runner, unmounts the projection, closes the exec bridge, disconnects the local agent connection, and preserves the stable workspace mapping.

## Build Artifacts

Build the MVP binaries from the repository root:

```bash
make build-ccgo
```

This writes:

- `backend/bin/ccgo`: local user CLI and bundled local-agent runner.
- `backend/bin/ccgo-wrapper`: server-side `CLAUDE_CODE_SHELL_PREFIX` wrapper.

For a specific server architecture, run the backend target with normal Go cross-build variables, for example:

```bash
GOOS=linux GOARCH=amd64 make -C backend ccgo-binaries
```

## Architecture

```text
local machine
  ccgo CLI + local ccgo-agent
  canonical project root
  local shell/git/npm/docker/tests

platform server
  Claude Code process
  stable projection root
  FUSE projection backed by local-agent file RPC
  ccgo shell wrapper backed by local-agent exec RPC
```

No user system `sshd`, SSHFS, manual tunnel, or manual mount is part of the product path.

## Login Flow

```text
ccgo login
  -> POST /api/v1/ccgo/device-login/start
  -> user approves /ccgo/device?code=ABCD-EFGH in the browser
  -> POST /api/v1/ccgo/device-login/poll
  -> local config stores the returned platform token
```

The approval page requires a normal authenticated browser session. If the user is not logged in, the frontend auth guard redirects to login and then returns to the device approval route.

## Path Semantics

- One user plus one canonical local root maps to one stable server projection root.
- Workstation runs are disposable process lifecycle records; they are not workspace identity.
- Server roots must not be shown as the real workspace in normal user-facing copy.
- Generated ccgo instructions are written outside the projected project and passed to Claude Code with `--append-system-prompt-file`.
- User repository files such as `CLAUDE.md`, `.claude/CLAUDE.md`, and `CLAUDE.local.md` are preserved as projected local files.

## Server Prerequisites

- Claude Code installed and authenticated in the managed server environment.
- FUSE support on runner hosts. Container deployments need `/dev/fuse` and the required capabilities configured explicitly.
- PTY support for the Claude Code TUI.
- A stable workspace base directory, for example `/var/lib/ccgo/workspaces`.
- A reachable `ccgo-wrapper` binary path for `CLAUDE_CODE_SHELL_PREFIX`.
- Sticky API/hub/projection/runner placement for MVP. Brokered multi-runner scheduling is a later design.

## Local Prerequisites

- macOS/Linux: the bundled `ccgo` binary can start the local agent from the user's normal account. `bash` must be available on `PATH` because Claude Code shell commands are executed locally through `bash -lc`.
- Windows MVP: file/path metadata is modeled, but Bash command execution requires WSL or Git Bash with `bash.exe` on `PATH`. Native PowerShell semantics are deferred.
- `ccgo <local_path>` runs a local preflight before creating the workspace runner. It verifies the path is a directory and that the local Bash probe can execute inside that root. Failure stops startup before the server projection or Claude runner is started.

## Failure Semantics

`ccgo` fails closed. It must not silently read a stale server copy or execute commands on the server.

- Agent disconnected: file and command requests fail with `AGENT_DISCONNECTED`.
- Projection unavailable: workstation start fails with a projection error.
- Runner cleanup failure: `ccgo stop` returns `CCGO_RUNNER_CLEANUP_FAILED` and leaves state available for retry.
- Auth failure: workspace and lifecycle APIs return the platform auth error.
- Command timeout/failure: the wrapper returns the real exit code or timeout error to Claude Code.

## Audit Boundaries

Command audit records store metadata only: workspace, user, run, request id, command hash, redacted command text, mapped cwd, status, exit code, timing, and failure reason. Raw stdout and stderr are not stored by default.

Tokens, environment values, and command output must not be written to normal logs.

## MVP Completion Boundary

Implemented for the productized MVP:

- Token login and browser-approved device login.
- Stable user plus canonical-local-root workspace mapping.
- Outbound local agent over WebSocket; no system `sshd`, SSHFS, or manual tunnel.
- Server FUSE projection backed by local file RPC, including stat, read, write, list, mkdir, remove, rename, truncate, chmod, and readlink.
- Server-side Claude Code launch with isolated config and generated append-system-prompt guidance.
- `CLAUDE_CODE_SHELL_PREFIX` wrapper that maps server projection paths back to local paths and executes through the local agent.
- `status`, `stop`, local preflight, `doctor`, command audit metadata, and `make build-ccgo` artifacts.
- A local integration harness that verifies projected file writes and wrapper-routed commands both affect the local project root.

Deferred beyond MVP:

- Native PowerShell command semantics for Windows. The MVP requires WSL or Git Bash for command execution.
- Dangerous-command approval UI, policy authoring UI, and per-command human approval.
- Multi-runner scheduling across non-sticky workers.
- Large-repository performance tuning and any future sync-acceleration mode.
- A full browser dashboard beyond login/device approval and basic lifecycle status.
