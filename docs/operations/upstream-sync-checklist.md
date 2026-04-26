# Upstream Sync Checklist

Follow this checklist when merging changes from the official `Wei-Shaw/sub2api` repository into our private fork.

## 1. Preparation
- [ ] Ensure local `main` and `private-deploy` branches are clean (`git status`).
- [ ] Fetch latest from upstream: `git fetch upstream`.

## 2. Sync Upstream to Main
- [ ] Checkout main: `git checkout main`.
- [ ] Merge upstream: `git merge upstream/main`.
- [ ] Resolve any conflicts (rare on `main`).
- [ ] Run basic tests: `cd backend && go test -tags=unit ./...`.
- [ ] Push to private origin: `git push origin main`.

## 3. Merge Main to Private Deploy
- [ ] Checkout private-deploy: `git checkout private-deploy`.
- [ ] Merge main: `git merge main`.
- [ ] **Conflict Resolution:**
    - [ ] Review changes in `deploy/`. Ensure our private overlays (`docker-compose.private.yml`) and scripts are not accidentally deleted or broken.
    - [ ] Review `go.mod` and `pnpm-lock.yaml`.
    - [ ] If core logic was changed upstream, ensure it doesn't break our private features.
- [ ] Run full test suite:
    - [ ] Backend: `cd backend && go test -tags=unit,integration ./...`
    - [ ] Frontend: `cd frontend && pnpm build` (verifies buildability).
- [ ] Push to private origin: `git push origin private-deploy`.

## 4. Local Build Validation
- [ ] Run `deploy/build_image.sh` to ensure the combined codebase still builds a valid Docker image.
- [ ] (Optional) Start the stack locally using the private override to smoke test:
    ```bash
    docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.private.yml up -d
    ```

## 5. Documentation Update
- [ ] If upstream added new configuration options, update `deploy/private/.env.production.example`.
- [ ] If upstream changed the deployment structure, update `docs/operations/deployment-layout.md`.

## 6. Readiness for Release
- [ ] Tag the release in the private repo (e.g., `v1.2.3-private.1`).
- [ ] Proceed to `deploy/scripts/export_release_bundle.sh`.
