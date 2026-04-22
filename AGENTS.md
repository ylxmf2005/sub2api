# Local Notes

- IP: `154.26.179.199`
- Key archive: `/Users/ethan/Desktop/s2a/DMIT-5FilTo7Lwa-ed25519.zip`
- Treat the zip archive above as the key.

# Release Versioning Rule

- For upstream-backed releases, default to the official upstream version tag (for example `v0.1.115`) as the deployed `SUB2API_RELEASE_TAG` and image version string.
- Do not invent custom deployment version names when an official upstream release tag already exists, unless the user explicitly asks for a private suffix.
- Keep the Docker/image release tag and the app's internal version separate: release tags may use the upstream `v` prefix, but the version injected into the binary/UI must stay in canonical numeric form without the leading `v` (for example `0.1.115`).
- When deploying local/private changes on top of an official upstream release, use the official tag plus a clear private suffix for the image/release tag (for example `v0.1.115-local1`) while keeping the app's internal version on the upstream canonical value (for example `0.1.115`).
