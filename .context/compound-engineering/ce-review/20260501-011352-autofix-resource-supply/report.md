# CE Review Autofix Report

Run ID: `20260501-011352-autofix-resource-supply`
Mode: `autofix`
Base: `32b354ced9c1c5db4a1302639950fdee6fcc1481`
Branch: `codex/settlement-cycle-participation`

## Scope

- Primary review focus: resource supply credit changes, especially self-service gating, OAuth UI typing, billing/reward safety, and debug-first behavior.
- Formal tracked diff is large and includes earlier settlement-cycle work on this branch.
- Important limitation: many resource supply implementation files are still untracked, so future PR tooling will need them added/staged for a clean formal diff. This review still manually included the untracked resource supply backend/frontend files.

## Applied Autofixes

1. Kept the user resource supply page visible in the sidebar even when self-service intake is disabled. The self-service setting now gates only the submit controls/API behavior, so users can still inspect balances, ledger history, and transfers.
   - `frontend/src/components/layout/AppSidebar.vue`

2. Fixed Vue template ref usage in the resource supply OAuth flow by destructuring composable refs into top-level bindings. This resolves the `vue-tsc` failures caused by passing nested `Ref` objects to template attributes.
   - `frontend/src/views/user/ResourceSupplyView.vue`

3. Removed a silent error swallow when a self-service account connection test fails. If marking the account rejected fails, the service now returns that update error explicitly.
   - `backend/internal/service/resource_supply_service.go`

4. Fixed API-key auth cache snapshots so resource supply reward settings survive cached authentication. Cached API keys now preserve group reward enabled state, multiplier, and self-service review policy.
   - `backend/internal/service/api_key_auth_cache.go`
   - `backend/internal/service/api_key_auth_cache_impl.go`
   - `backend/internal/repository/api_key_repo.go`
   - `backend/internal/service/api_key_service_cache_test.go`

5. Fixed resource supply account action idempotency payloads so the same idempotency key cannot replay an action response across different account IDs.
   - `backend/internal/handler/resource_supply_handler.go`
   - `backend/internal/handler/admin/resource_supply_handler.go`

6. Sanitized user-facing resource supply ledger responses so user endpoints no longer expose internal audit identifiers or other users' emails.
   - `backend/internal/service/resource_supply_service.go`
   - `backend/internal/service/resource_supply_service_test.go`

7. Added cleanup for failed self-service account group binding so a partially created supply account is not left behind when binding fails.
   - `backend/internal/service/resource_supply_service.go`

8. Aligned the Ent resource supply ledger schema with the migration's partial unique reward-event index and regenerated Ent code.
   - `backend/ent/schema/resource_supply_ledger.go`
   - `backend/ent/migrate/schema.go`

9. Blocked owner self-use rewards. Reward eligibility now requires the caller to be different from the supply account owner, and the repository checks the same invariant before ledger writes.
   - `backend/internal/service/gateway_service.go`
   - `backend/internal/repository/usage_billing_repo.go`
   - `backend/internal/service/resource_supply_service_test.go`

10. Restricted self-service OpenAI API-key `base_url` to the official OpenAI HTTPS host to avoid turning user-submitted supply accounts into arbitrary server-side outbound targets.
    - `backend/internal/service/resource_supply_service.go`
    - `backend/internal/service/resource_supply_service_test.go`

11. Made resource supply balance transfers verify that the user balance row was actually updated before committing the economic transfer.
    - `backend/internal/repository/resource_supply_repo.go`

12. Replaced hardcoded admin supply action labels/reasons/results with i18n keys.
    - `frontend/src/views/admin/AccountsView.vue`
    - `frontend/src/i18n/locales/en.ts`
    - `frontend/src/i18n/locales/zh.ts`

## Verification

- `git diff --check` passed.
- `go test ./...` passed in `backend/`.
- `npm run typecheck` passed in `frontend/`.
- `npm run build` passed in `frontend/`.
- Targeted service tests passed for API-key auth cache snapshots, resource supply ledger sanitization, reward command snapshots, self-use reward exclusion, and self-service OpenAI base URL validation.

## Residual Risk

- Add/stage the untracked resource supply files before PR creation so the formal review scope matches the implemented feature surface.
- Frontend build still reports existing Vite dynamic/static import and chunk-size warnings; the build completed successfully.
