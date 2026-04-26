# Private Fork Workflow

This document describes the workflow for maintaining and deploying a private fork of Sub2API.

## Overview

We maintain a private fork of [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) to carry local customizations, private deployment configurations, and specific features that are not intended for the upstream repository.

The goal is to keep our fork as close to upstream as possible, making it easy to pull in updates while maintaining a stable, locally-built production environment.

## Repository Roles

| Remote | URL | Role |
|---|---|---|
| `upstream` | `https://github.com/Wei-Shaw/sub2api.git` | Official source of truth. Read-only for us. |
| `origin` | (Private Repo URL) | Our private source of truth. Where we push our changes. |

## Branching Strategy

- `main`: Tracks `upstream/main`. Should ideally remain clean or only contain minimal, universally applicable changes.
- `private-deploy`: Our long-lived branch for production. Contains private deployment overlays (`deploy/docker-compose.private.yml`), migration scripts, and any private features.
- `feature/*`: Short-lived branches for developing new features or fixes, branched from `private-deploy` or `main`.

## Maintenance Loop (Syncing with Upstream)

The sync process should happen regularly to avoid large, painful merge conflicts.

1.  **Fetch Upstream Changes:**
    ```bash
    git fetch upstream
    ```

2.  **Update Local `main`:**
    ```bash
    git checkout main
    git merge upstream/main
    git push origin main
    ```

3.  **Merge into `private-deploy`:**
    ```bash
    git checkout private-deploy
    git merge main
    # Resolve any conflicts. Prefer keeping private deployment files intact.
    git push origin private-deploy
    ```

## Local Build & Release Process

We build artifacts locally and transfer them to the server as image bundles.

1.  **Merge & Verify:** Ensure `private-deploy` is synced and tests pass.
2.  **Build Image:** Use `deploy/build_image.sh` to create the Docker image.
3.  **Export Bundle:** Use `deploy/scripts/export_release_bundle.sh` to create a versioned `.tar.gz` bundle.
4.  **Transfer:** Ship the bundle to the production server.
5.  **Deploy:** Load the image and restart the Compose stack using the private overlay.

Detailed instructions for build and release can be found in `docs/operations/local-build-and-release.md`.

## Guidelines for Private Changes

- **Additive Overlays:** Prefer adding new files (e.g., `docker-compose.private.yml`) over modifying upstream files (e.g., `docker-compose.yml`).
- **Isolation:** Keep private scripts in `deploy/scripts/`, `deploy/migration/`, and `deploy/runbooks/`.
- **Documentation:** Document every private customization in `docs/operations/`.
- **No Secrets in Repo:** Never commit real secrets. Use `.env.production.example` as a template and keep the real `.env` only on the production server.

## Upstream Sync Checklist

Before every production cutover or major update, follow the [Upstream Sync Checklist](../operations/upstream-sync-checklist.md).
