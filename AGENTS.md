# Local Notes

- IP: `154.26.179.199`
- Key archive: `/Users/ethan/Desktop/s2a/DMIT-5FilTo7Lwa-ed25519.zip`
- Treat the zip archive above as the key.
- Private keys were moved out of this file into local ignored storage under `.codex-local/secrets/`.
- Never store private keys directly in `AGENTS.md`.

# Release Versioning Rule

- For upstream-backed releases, default to the official upstream version tag (for example `v0.1.115`) as the deployed `SUB2API_RELEASE_TAG` and image version string.
- Do not invent custom deployment version names or private suffixes when an official upstream release tag already exists, unless the user explicitly asks for one.
- Keep the Docker/image release tag and the app's internal version separate: release tags may use the upstream `v` prefix, but the version injected into the binary/UI must stay in canonical numeric form without the leading `v` (for example `0.1.115`).
- When deploying local/private changes on top of an official upstream release, still use the pure official tag by default (for example `v0.1.115`) while keeping the app's internal version on the upstream canonical value (for example `0.1.115`).

# Private Fork Release Workflow

- Official upstream releases must be merged into the local private fork before deployment; do not replace the private fork with the public upstream image.
- Build production images locally from the private fork, export a release bundle, and upload that already-built bundle to the server.
- Do not build Sub2API on the production server. The server should only load the prepared Docker image bundle, update `.env`, restart Compose, and run smoke checks.
- Do not start an upstream release merge while the local worktree has uncommitted production-relevant changes. Inspect `git status --short`, classify every modified/untracked file, and commit production-relevant private work first.
- When the main worktree has unrelated or explicitly excluded in-progress changes, create an isolated clean build directory/worktree from the committed private fork state after the production-relevant work is committed.

# One-Shot Production Update SOP

Sub2API powers the AI that is doing the deployment work. Treat each production restart as a one-shot operation: do not touch the server runtime until the release candidate has been proven locally.

Required release source rule:

- The release candidate must be built from the private fork state that production actually depends on, not from a stale committed base.
- If production already has database migrations or behavior from local private work, that private work must be committed locally and included in the release candidate before merging the upstream tag.
- A dirty worktree is a release blocker by default. Continue only after the dirty files are either committed, reverted by explicit user request, or documented as unrelated/excluded from the release.
- Upstream updates are incremental: preserve private features and migrations, merge the upstream changes into them, and adapt private code to upstream bug fixes or API changes. Never "simplify" an update by dropping private migrations, private services, or private access checks.
- Before building, write down the source tuple in the working notes: private base commit, dirty-worktree decision, upstream tag, merge commit, app version.

Pre-merge checklist:

- Fetch upstream tags and confirm the requested official tag exists.
- Run `git status --short --branch`.
- Review modified and untracked files. For each file, decide whether it is production-relevant, unrelated local work, or generated/temporary output.
- Commit production-relevant private changes before the upstream merge. Do not build from an uncommitted private patch unless the user explicitly asks for that emergency path.
- Create the release integration worktree from that committed private base.
- Merge the official upstream tag into the private integration worktree.
- Review upstream changes by feature area, not only by conflict count, and check whether upstream bug fixes must be adapted to private code paths.

Local acceptance gates before any server mutation:

- Merge the upstream tag into an isolated local build directory/worktree that already contains the required private production state.
- Resolve generated-code conflicts from the source providers, then regenerate generated files instead of choosing either side by hand.
- Confirm generated dependency wiring keeps both upstream additions and private dependencies.
- Run `git diff --check`.
- Run backend tests at minimum with `go test ./...` from `backend/`.
- If the release includes or depends on migrations, restore or connect a local clone of production data and verify migrations plus the affected queries against that clone.
- For settlement-pool releases, explicitly verify that candidate users, current-cycle participants, and API-key auth agree with the production data shape.
- Build the Docker image locally with the official upstream tag as the image tag and canonical numeric app version.
- Run the built image locally, or against a local production-data clone, and smoke test `/health` plus one real AI request path that uses the production-critical pool/subscription mode.
- Inspect the built image version before exporting the bundle.
- Treat `/health` as necessary but insufficient. The release is not accepted until the same auth/billing mode used by the deployment AI can complete a real request.

Server cutover rules:

- Do not run `docker build` on the server.
- Do not run ad hoc Compose commands against a single override file. Use the same Compose file set as the deployment scripts: `docker-compose.local.yml` plus `docker-compose.private.yml` with the deployment `.env`.
- Upload only the already-built release bundle and deployment scripts needed for loading/cutover.
- Back up before cutover when schema migrations, persistent data, or irreversible state changes are involved. Keep backups boring and minimal: `.env`, Compose files, PostgreSQL dump, and runtime data excluding live log files.
- A rollback image/tag may be prepared, but the goal is not to rely on rollback. If local acceptance cannot prove that the AI request path will survive the restart, do not deploy.
- After cutover, verify container health, app version, and the real AI request path before considering the deployment complete.
- If any server cutover command fails or produces an unexpected Compose project/container name, stop and diagnose from another control path. Do not improvise additional server mutations.

# v0.1.120 Incident Note

- The failed v0.1.120 attempt was caused by building from a stale committed private base while production already depended on newer private settlement-cycle participation work.
- Production had already applied `134_settlement_pool_cycle_participation.sql`, splitting settlement-pool access into candidates and current-cycle participants.
- The first v0.1.120 image did not include the corresponding private code, so API-key auth checked settlement-pool participation through the wrong code path and returned `SETTLEMENT_POOL_PARTICIPANT_REQUIRED`.
- The upstream update itself was not the root problem. The release assembly process was wrong: the release candidate did not contain the full private production state before the upstream tag was merged.
