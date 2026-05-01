/**
 * Admin Resource Supply API endpoints
 * Handles resource supply ledger, account actions, and adjustments
 */

import { apiClient } from '../client'
import type { Account, ResourceSupplyLedgerEntry, PaginatedResponse } from '@/types'

function idempotencyHeaders(): { 'Idempotency-Key': string } {
  return { 'Idempotency-Key': crypto.randomUUID() }
}

/**
 * List resource supply ledger entries with pagination and filters
 * @param params - Query parameters for filtering and pagination
 * @returns Paginated list of ledger entries
 */
export async function getLedger(params?: {
  owner_user_id?: number
  caller_user_id?: number
  group_id?: number
  account_id?: number
  type?: string
  request_id?: string
  page?: number
  page_size?: number
}): Promise<PaginatedResponse<ResourceSupplyLedgerEntry>> {
  const { data } = await apiClient.get<PaginatedResponse<ResourceSupplyLedgerEntry>>(
    '/admin/resource-supply/ledger',
    { params }
  )
  return data
}

/**
 * Approve a resource supply account
 * @param id - Account ID
 * @param reason - Approval reason
 */
export async function approveAccount(
  id: number,
  reason: string
): Promise<Account> {
  const { data } = await apiClient.post<Account>(
    `/admin/resource-supply/accounts/${id}/approve`,
    { reason },
    { headers: idempotencyHeaders() }
  )
  return data
}

/**
 * Reject a resource supply account
 * @param id - Account ID
 * @param reason - Rejection reason
 */
export async function rejectAccount(
  id: number,
  reason: string
): Promise<Account> {
  const { data } = await apiClient.post<Account>(
    `/admin/resource-supply/accounts/${id}/reject`,
    { reason },
    { headers: idempotencyHeaders() }
  )
  return data
}

/**
 * Pause a resource supply account
 * @param id - Account ID
 * @param reason - Pause reason
 */
export async function pauseAccount(
  id: number,
  reason: string
): Promise<Account> {
  const { data } = await apiClient.post<Account>(
    `/admin/resource-supply/accounts/${id}/pause`,
    { reason },
    { headers: idempotencyHeaders() }
  )
  return data
}

/**
 * Resume a resource supply account
 * @param id - Account ID
 * @param reason - Resume reason
 */
export async function resumeAccount(
  id: number,
  reason: string
): Promise<Account> {
  const { data } = await apiClient.post<Account>(
    `/admin/resource-supply/accounts/${id}/resume`,
    { reason },
    { headers: idempotencyHeaders() }
  )
  return data
}

/**
 * Create a resource supply adjustment
 * @param adjustment - Adjustment details
 * @returns Created adjustment confirmation
 */
export async function createAdjustment(adjustment: {
  owner_user_id: number
  amount: number
  note: string
}): Promise<ResourceSupplyLedgerEntry> {
  const { data } = await apiClient.post<ResourceSupplyLedgerEntry>(
    '/admin/resource-supply/adjustments',
    adjustment,
    { headers: idempotencyHeaders() }
  )
  return data
}

export const resourceSupplyAPI = {
  getLedger,
  approveAccount,
  rejectAccount,
  pauseAccount,
  resumeAccount,
  createAdjustment
}

export default resourceSupplyAPI
