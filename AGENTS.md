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
