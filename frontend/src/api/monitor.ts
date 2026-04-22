import { apiClient } from './client'
import type { AdminGroup, UserSpendingRankingResponse, UserBreakdownItem } from '@/types'
import type {
  DashboardSnapshotV2Params,
  DashboardSnapshotV2Response,
  UserBreakdownParams,
  UserBreakdownResponse,
  UserTrendParams,
  UserTrendResponse,
  UserSpendingRankingParams
} from './admin/dashboard'

export async function getSnapshotV2(
  params?: DashboardSnapshotV2Params
): Promise<DashboardSnapshotV2Response> {
  const { data } = await apiClient.get<DashboardSnapshotV2Response>('/monitor/snapshot-v2', {
    params
  })
  return data
}

export async function getUserUsageTrend(params?: UserTrendParams): Promise<UserTrendResponse> {
  const { data } = await apiClient.get<UserTrendResponse>('/monitor/users-trend', {
    params
  })
  return data
}

export async function getUserSpendingRanking(
  params?: UserSpendingRankingParams
): Promise<UserSpendingRankingResponse> {
  const { data } = await apiClient.get<UserSpendingRankingResponse>('/monitor/users-ranking', {
    params
  })
  return data
}

export async function getUserBreakdown(params: UserBreakdownParams): Promise<UserBreakdownResponse> {
  const { data } = await apiClient.get<UserBreakdownResponse>('/monitor/user-breakdown', {
    params
  })
  return data
}

export async function getAllGroups(): Promise<AdminGroup[]> {
  const { data } = await apiClient.get<AdminGroup[]>('/monitor/groups/all')
  return data
}

export const monitorAPI = {
  dashboard: {
    getSnapshotV2,
    getUserUsageTrend,
    getUserSpendingRanking,
    getUserBreakdown
  },
  groups: {
    getAll: getAllGroups
  }
}

export default monitorAPI

export type { UserBreakdownItem }
