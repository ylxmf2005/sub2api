import { apiClient } from './client'
import type { SettlementPoolSummary } from '@/types'

export async function list(): Promise<SettlementPoolSummary[]> {
  const { data } = await apiClient.get<SettlementPoolSummary[]>('/settlement-pools')
  return data
}

export async function getByGroup(groupId: number): Promise<SettlementPoolSummary> {
  const { data } = await apiClient.get<SettlementPoolSummary>(`/settlement-pools/${groupId}`)
  return data
}

export async function joinCurrentCycle(groupId: number): Promise<SettlementPoolSummary> {
  const { data } = await apiClient.post<SettlementPoolSummary>(`/settlement-pools/${groupId}/participation`)
  return data
}

export default {
  list,
  getByGroup,
  joinCurrentCycle
}
