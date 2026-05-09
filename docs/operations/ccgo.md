# ccgo Reverse Workstation Operations

`ccgo` runs Claude Code on the platform while keeping the user's project files and developer commands on the user's machine.

The product invariant is strict: the canonical workspace is the user's local project root. The server workspace root is only a stable projection directory so Claude Code can bind its own project context and session history to a concrete directory.

## User Commands

```bash
ccgo login
ccgo login <token>
ccgo .
ccgo status
ccgo stop
```

- `ccgo login` starts the device-code login flow, opens or prints the approval URL, polls until approval, and stores the issued platform token in the user's OS config directory.
- `ccgo login <token>` stores the platform token in the user's OS config directory.
- `ccgo <local_path>` resolves the canonical local path, creates or reuses the stable workspace mapping, starts the local agent, asks the server to start Claude Code, and records the last workspace id for `status` and `stop`.
- `ccgo status [workspace_id]` reports the workspace mapping, local-agent connection, projection/runner health, and latest workstation run.
- `ccgo stop [workspace_id]` stops the server runner, unmounts the projection, closes the exec bridge, disconnects the local agent connection, and preserves the stable workspace mapping.

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

- macOS/Linux: the bundled `ccgo` binary can start the local agent from the user's normal account.
- Windows MVP: file/path metadata is modeled, but Bash command execution requires WSL or Git Bash. Native PowerShell semantics are deferred.

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
