/**
 * System API endpoints for admin operations
 */

import { apiClient } from '../client'

export interface ReleaseInfo {
  name: string
  body: string
  published_at: string
  html_url: string
}

export interface VersionInfo {
  current_version: string
  latest_version: string
  has_update: boolean
  release_info?: ReleaseInfo
  cached: boolean
  warning?: string
  build_type: string // "source" for manual builds, "release" for CI builds
}

/**
 * Get current version
 */
export async function getVersion(): Promise<{ version: string }> {
  const { data } = await apiClient.get<{ version: string }>('/admin/system/version')
  return data
}

/**
 * Check for updates
 * @param force - Force refresh from GitHub API
 */
export async function checkUpdates(force = false): Promise<VersionInfo> {
  const { data } = await apiClient.get<VersionInfo>('/admin/system/check-updates', {
    params: force ? { force: 'true' } : undefined
  })
  return data
}

export interface UpdateResult {
  message: string
  need_restart: boolean
}

export interface UserIDMaintenanceStatus {
  max_user_id: number
  next_user_id: number
  sequence_name: string
}

export interface SetUserNextIDRequest {
  next_user_id: number
}

export interface SetUserNextIDResult {
  max_user_id: number
  previous_next_user_id: number
  next_user_id: number
}

export interface ChangeUserIDRequest {
  old_user_id: number
  new_user_id: number
  confirmation: string
}

export interface ChangeUserIDResult {
  old_user_id: number
  new_user_id: number
  updated_rows: number
  updated_by_target: Record<string, number>
  next_user_id: number
}

export interface UserIDMaintenanceOperationResponse<T> {
  operation_id: string
  result: T
}

export interface RollbackVersionInfo {
  version: string
  published_at: string
  html_url: string
}

/**
 * Get versions available for rollback (up to 3 versions older than current)
 */
export async function getRollbackVersions(): Promise<{ versions: RollbackVersionInfo[] }> {
  const { data } = await apiClient.get<{ versions: RollbackVersionInfo[] }>(
    '/admin/system/rollback-versions'
  )
  return data
}

/**
 * Perform system update
 * Downloads and applies the latest version
 */
export async function performUpdate(): Promise<UpdateResult> {
  const { data } = await apiClient.post<UpdateResult>('/admin/system/update')
  return data
}

/**
 * Rollback to a previous version
 * @param version - Target version (e.g. "0.1.146"); omit to restore the local backup binary
 */
export async function rollback(version?: string): Promise<UpdateResult> {
  const { data } = await apiClient.post<UpdateResult>(
    '/admin/system/rollback',
    version ? { version } : undefined
  )
  return data
}

/**
 * Restart the service
 */
export async function restartService(): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>('/admin/system/restart')
  return data
}

function idempotencyConfig(idempotencyKey?: string) {
  return idempotencyKey
    ? { headers: { 'Idempotency-Key': idempotencyKey } }
    : undefined
}

export async function getUserIDMaintenanceStatus(): Promise<UserIDMaintenanceStatus> {
  const { data } = await apiClient.get<UserIDMaintenanceStatus>('/admin/system/user-id-maintenance')
  return data
}

export async function setUserNextID(
  request: SetUserNextIDRequest,
  idempotencyKey?: string
): Promise<UserIDMaintenanceOperationResponse<SetUserNextIDResult>> {
  const { data } = await apiClient.post<UserIDMaintenanceOperationResponse<SetUserNextIDResult>>(
    '/admin/system/user-id-maintenance/next-id',
    request,
    idempotencyConfig(idempotencyKey)
  )
  return data
}

export async function changeUserID(
  request: ChangeUserIDRequest,
  idempotencyKey?: string
): Promise<UserIDMaintenanceOperationResponse<ChangeUserIDResult>> {
  const { data } = await apiClient.post<UserIDMaintenanceOperationResponse<ChangeUserIDResult>>(
    '/admin/system/user-id-maintenance/change-user-id',
    request,
    idempotencyConfig(idempotencyKey)
  )
  return data
}

export const systemAPI = {
  getVersion,
  checkUpdates,
  performUpdate,
  getRollbackVersions,
  rollback,
  restartService,
  getUserIDMaintenanceStatus,
  setUserNextID,
  changeUserID
}

export default systemAPI
