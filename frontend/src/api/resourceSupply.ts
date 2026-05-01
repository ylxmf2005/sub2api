/**
 * Resource Supply API endpoints
 * Handles user-facing resource supply management, accounts, and ledger
 */

import { apiClient } from './client'
import type {
  ResourceSupplySummary,
  ResourceSupplyLedgerResponse,
  ResourceSupplyTransferResult,
  ResourceSupplyOwnedAccount,
} from '@/types'

function idempotencyHeaders(): { 'Idempotency-Key': string } {
  return { 'Idempotency-Key': crypto.randomUUID() }
}

/**
 * Get the current user's resource supply summary
 * @returns Resource supply summary including balance, accounts, and recent ledger
 */
export async function getSummary(): Promise<ResourceSupplySummary> {
  const { data } = await apiClient.get<ResourceSupplySummary>('/user/resource-supply')
  return data
}

/**
 * Get paginated resource supply ledger entries
 * @param params - Query parameters for filtering and pagination
 * @returns Paginated ledger response
 */
export async function getLedger(params: {
  type?: string
  page?: number
  page_size?: number
}): Promise<ResourceSupplyLedgerResponse> {
  const { data } = await apiClient.get<ResourceSupplyLedgerResponse>(
    '/user/resource-supply/ledger',
    { params },
  )
  return data
}

/**
 * Register a new account by providing an OpenAI-compatible API key
 * @param payload - Account details including group, name, API key, base URL, and model
 * @returns The newly created owned account
 */
export async function addOpenAIApiKeyAccount(payload: {
  group_id: number
  name?: string
  api_key: string
  base_url?: string
  model_id?: string
}): Promise<ResourceSupplyOwnedAccount> {
  const { data } = await apiClient.post<ResourceSupplyOwnedAccount>(
    '/user/resource-supply/accounts/openai-api-key',
    payload,
    { headers: idempotencyHeaders() },
  )
  return data
}

/**
 * Generate an OAuth authorization URL for linking an OpenAI account
 * @param payload - Redirect URI and proxy ID
 * @returns Object containing the generated authorization URL
 */
export async function generateOpenAIAuthUrl(payload: {
  redirect_uri?: string
  proxy_id?: number | null
}): Promise<{ auth_url: string; session_id: string }> {
  const { data } = await apiClient.post<{ auth_url: string; session_id: string }>(
    '/user/resource-supply/openai/generate-auth-url',
    payload,
  )
  return data
}

/**
 * Exchange an OAuth authorization code for an OpenAI-linked account
 * @param payload - OAuth exchange details
 * @returns The newly created owned account
 */
export async function exchangeOpenAICode(payload: {
  group_id: number
  name?: string
  session_id: string
  code: string
  state: string
  redirect_uri?: string
  proxy_id?: number | null
  model_id?: string
}): Promise<ResourceSupplyOwnedAccount> {
  const { data } = await apiClient.post<ResourceSupplyOwnedAccount>(
    '/user/resource-supply/openai/exchange-code',
    payload,
    { headers: idempotencyHeaders() },
  )
  return data
}

/**
 * Pause a resource supply account
 * @param id - Account ID
 * @param reason - Reason for pausing
 * @returns The updated owned account
 */
export async function pauseAccount(
  id: number,
  reason: string,
): Promise<ResourceSupplyOwnedAccount> {
  const { data } = await apiClient.post<ResourceSupplyOwnedAccount>(
    `/user/resource-supply/accounts/${id}/pause`,
    { reason },
    { headers: idempotencyHeaders() },
  )
  return data
}

/**
 * Revoke a resource supply account
 * @param id - Account ID
 * @param reason - Reason for revoking
 * @returns The updated owned account
 */
export async function revokeAccount(
  id: number,
  reason: string,
): Promise<ResourceSupplyOwnedAccount> {
  const { data } = await apiClient.post<ResourceSupplyOwnedAccount>(
    `/user/resource-supply/accounts/${id}/revoke`,
    { reason },
    { headers: idempotencyHeaders() },
  )
  return data
}

/**
 * Transfer resource supply balance to user balance
 * Each request uses a unique idempotency key to prevent duplicate transfers
 * @returns Transfer result with amount and new balance
 */
export async function transfer(): Promise<ResourceSupplyTransferResult> {
  const { data } = await apiClient.post<ResourceSupplyTransferResult>(
    '/user/resource-supply/transfer',
    undefined,
    {
      headers: idempotencyHeaders(),
    },
  )
  return data
}

export const resourceSupplyAPI = {
  getSummary,
  getLedger,
  addOpenAIApiKeyAccount,
  generateOpenAIAuthUrl,
  exchangeOpenAICode,
  pauseAccount,
  revokeAccount,
  transfer,
}

export default resourceSupplyAPI
