import { apiClient } from '../client'
import type { SettlementPoolSummary, SettlementPoolTier } from '@/types'

export interface UpdateSettlementPoolConfigRequest {
  total_cost: number
  base_ratio: number
  market_cap: number
  tiers: SettlementPoolTier[]
}

export async function getSummary(groupId: number): Promise<SettlementPoolSummary> {
  const { data } = await apiClient.get<SettlementPoolSummary>(`/admin/settlement-pools/groups/${groupId}`)
  return data
}

export async function updateConfig(
  groupId: number,
  payload: UpdateSettlementPoolConfigRequest
): Promise<SettlementPoolSummary> {
  const { data } = await apiClient.put<SettlementPoolSummary>(
    `/admin/settlement-pools/groups/${groupId}/config`,
    payload
  )
  return data
}

export async function syncCandidates(
  groupId: number,
  userIds: number[]
): Promise<SettlementPoolSummary> {
  const { data } = await apiClient.put<SettlementPoolSummary>(
    `/admin/settlement-pools/groups/${groupId}/candidates`,
    { user_ids: userIds }
  )
  return data
}

export async function forceJoinCurrentCycle(
  groupId: number,
  userIds: number[]
): Promise<SettlementPoolSummary> {
  const { data } = await apiClient.post<SettlementPoolSummary>(
    `/admin/settlement-pools/groups/${groupId}/participants`,
    { user_ids: userIds }
  )
  return data
}

export async function removeCurrentParticipant(
  groupId: number,
  userId: number
): Promise<SettlementPoolSummary> {
  const { data } = await apiClient.delete<SettlementPoolSummary>(
    `/admin/settlement-pools/groups/${groupId}/participants/${userId}`
  )
  return data
}

export async function startNextCycle(groupId: number): Promise<SettlementPoolSummary> {
  const { data } = await apiClient.post<SettlementPoolSummary>(
    `/admin/settlement-pools/groups/${groupId}/start-cycle`
  )
  return data
}

export default {
  getSummary,
  updateConfig,
  syncCandidates,
  forceJoinCurrentCycle,
  removeCurrentParticipant,
  startNextCycle
}
