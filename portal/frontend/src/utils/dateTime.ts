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

const EMPTY_DATE_PLACEHOLDER = '-'
const EPOCH_MILLISECONDS_CUTOFF = 100000000000

export function toEpochMilliseconds(epochTimestamp: number | null | undefined): number | null {
  if (epochTimestamp == null || !Number.isFinite(epochTimestamp)) {
    return null
  }

  return epochTimestamp < EPOCH_MILLISECONDS_CUTOFF ? epochTimestamp * 1000 : epochTimestamp
}

export function toStartOfDayEpochMilliseconds(dateText: string): number | undefined {
  if (!dateText) {
    return undefined
  }

  const epochMilliseconds = new Date(`${dateText}T00:00:00`).getTime()

  return Number.isNaN(epochMilliseconds) ? undefined : epochMilliseconds
}

export function toEndOfDayEpochMilliseconds(dateText: string): number | undefined {
  if (!dateText) {
    return undefined
  }

  const epochMilliseconds = new Date(`${dateText}T23:59:59`).getTime()

  return Number.isNaN(epochMilliseconds) ? undefined : epochMilliseconds
}

export function formatEpochTimestamp(
  epochTimestamp: number | null | undefined,
  options?: Intl.DateTimeFormatOptions,
  locales?: Intl.LocalesArgument,
): string {
  const epochMilliseconds = toEpochMilliseconds(epochTimestamp)

  if (epochMilliseconds == null) {
    return EMPTY_DATE_PLACEHOLDER
  }

  return new Date(epochMilliseconds).toLocaleString(locales, options)
}

export function formatIsoDateTime(
  dateTimeText: string | null | undefined,
  options?: Intl.DateTimeFormatOptions,
  locales?: Intl.LocalesArgument,
): string {
  if (!dateTimeText) {
    return EMPTY_DATE_PLACEHOLDER
  }

  const parsedDate = new Date(dateTimeText)

  if (Number.isNaN(parsedDate.getTime())) {
    return EMPTY_DATE_PLACEHOLDER
  }

  return parsedDate.toLocaleString(locales, options)
}
