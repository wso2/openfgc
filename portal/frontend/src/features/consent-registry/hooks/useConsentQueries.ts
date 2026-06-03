/*
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import {
  keepPreviousData,
  type UseMutationResult,
  type UseQueryResult,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import {
  approveMyConsent,
  fetchMyConsentByID,
  fetchMyConsents,
  revokeMyConsent,
} from '../api/consentsApi'
import {
  isConsentApprovableStatus,
  isConsentRevokableStatus,
  normalizeConsentStatus,
} from '../utils/statusChip'
import type {
  ConsentApprovalSelection,
  ConsentDetailAPI,
  ConsentListQueryParams,
  ConsentRecord,
  ConsentRegistryFilters,
} from '../../../types/consent'
import { isConsentAPIStatus } from '../../../types/consent'
import {
  toEndOfDayEpochMilliseconds,
  toEpochMilliseconds,
  toStartOfDayEpochMilliseconds,
} from '../../../utils/dateTime'

interface ConsentListResult {
  rows: ConsentRecord[]
  total: number
}

interface ApproveConsentVariables {
  consentID: string
  selectedOptionalElements: ConsentApprovalSelection[]
}

function toListParams(
  filters: ConsentRegistryFilters,
  page: number,
  rowsPerPage: number,
): ConsentListQueryParams {
  const statusFilterMap: Record<Exclude<ConsentRegistryFilters['status'], 'All'>, string> = {
    Active: 'ACTIVE',
    Pending: 'CREATED',
    Rejected: 'REJECTED',
    Revoked: 'REVOKED',
    Expired: 'EXPIRED',
  }

  return {
    consentStatuses: filters.status === 'All' ? undefined : statusFilterMap[filters.status],
    consentTypes: filters.consentType.trim() || undefined,
    fromTime: toStartOfDayEpochMilliseconds(filters.startDate),
    toTime: toEndOfDayEpochMilliseconds(filters.endDate),
    limit: rowsPerPage,
    offset: page * rowsPerPage,
  }
}

function toConsentRow(consent: ConsentDetailAPI): ConsentRecord {
  const normalizedStatus = normalizeConsentStatus(consent.status)

  if (!isConsentAPIStatus(normalizedStatus)) {
    throw new Error(`Unsupported consent status received from API: ${consent.status}`)
  }

  return {
    id: consent.id,
    clientName: consent.clientId,
    type: consent.type,
    status: normalizedStatus,
    purposes: consent.purposes.map((purpose) => purpose.name),
    updatedAt: new Date(toEpochMilliseconds(consent.updatedTime) ?? 0).toISOString(),
    expirationTime: consent.validityTime ?? 0,
    canRevoke: isConsentRevokableStatus(normalizedStatus),
    canApprove: isConsentApprovableStatus(normalizedStatus),
  }
}

export function useConsentListQuery(
  filters: ConsentRegistryFilters,
  page: number,
  rowsPerPage: number,
): UseQueryResult<ConsentListResult> {
  const params = toListParams(filters, page, rowsPerPage)

  return useQuery<ConsentListResult>({
    queryKey: ['consents', params],
    queryFn: async (): Promise<ConsentListResult> => {
      const response = await fetchMyConsents(params)
      return {
        rows: response.data.map(toConsentRow),
        total: response.metadata.total,
      }
    },
    placeholderData: keepPreviousData,
  })
}

export function useConsentDetailQuery(
  consentID: string | undefined,
): UseQueryResult<ConsentDetailAPI> {
  return useQuery<ConsentDetailAPI>({
    queryKey: ['consent', consentID],
    queryFn: async (): Promise<ConsentDetailAPI> => fetchMyConsentByID(String(consentID)),
    enabled: Boolean(consentID),
  })
}

export function useApproveConsentMutation(): UseMutationResult<
  unknown,
  Error,
  ApproveConsentVariables
> {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      consentID,
      selectedOptionalElements,
    }: ApproveConsentVariables): Promise<unknown> =>
      approveMyConsent(consentID, selectedOptionalElements),
    onSuccess: async (_data, variables): Promise<void> => {
      await queryClient.invalidateQueries({ queryKey: ['consents'] })
      await queryClient.invalidateQueries({ queryKey: ['consent', variables.consentID] })
    },
  })
}

export function useRevokeConsentMutation(): UseMutationResult<unknown, Error, string> {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (consentID: string): Promise<unknown> => revokeMyConsent(consentID),
    onSuccess: async (_data, consentID): Promise<void> => {
      await queryClient.invalidateQueries({ queryKey: ['consents'] })
      await queryClient.invalidateQueries({ queryKey: ['consent', consentID] })
    },
  })
}
