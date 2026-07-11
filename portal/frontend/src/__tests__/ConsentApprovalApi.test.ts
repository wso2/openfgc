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

import { afterEach, describe, expect, it, vi } from 'vitest'
import { clearAuthBridge, registerAuthBridge } from '../auth/authBridge'
import { approveMyConsent } from '../features/consent-registry/api/consentsApi'
import { apiRequest } from '../utils/apiClient'

const fetchMock = vi.fn()

afterEach(() => {
  fetchMock.mockReset()
  vi.unstubAllGlobals()
})

describe('approveMyConsent', () => {
  it('sends selected optional approvals to the BFF approve endpoint', async () => {
    vi.stubGlobal('fetch', fetchMock)
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({}),
    })

    const selectedOptionalElements = [{ purposeName: 'profile_access', elementName: 'last_name' }]

    await approveMyConsent('consent/123?draft', selectedOptionalElements)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [requestUrl, requestInit] = fetchMock.mock.calls[0] ?? []
    const requestHeaders = new Headers((requestInit?.headers as HeadersInit | undefined) ?? {})

    expect(String(requestUrl)).toContain('/me/consents/consent%2F123%3Fdraft/approve')
    expect(requestInit).toMatchObject({
      method: 'POST',
      credentials: 'omit',
      body: JSON.stringify(selectedOptionalElements),
    })
    expect(requestHeaders.get('Accept')).toBe('application/json')
    expect(requestHeaders.get('Content-Type')).toBe('application/json')
  })
})

describe('apiRequest', () => {
  it('rejects successful responses without a JSON body', async () => {
    vi.stubGlobal('fetch', fetchMock)
    fetchMock.mockResolvedValue({
      ok: true,
      status: 204,
    })

    await expect(apiRequest<unknown>('/empty')).rejects.toThrow('Use apiRequestNoContent instead')
  })

  it('rejects absolute paths before sending a request', async () => {
    vi.stubGlobal('fetch', fetchMock)

    await expect(apiRequest<unknown>('https://example.com/consents')).rejects.toThrow(
      'apiClient path must be relative',
    )
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('attaches the SDK-managed bearer token when an auth provider is registered', async () => {
    vi.stubGlobal('fetch', fetchMock)
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({}),
    })
    const provider = vi.fn(async () => 'access-token')
    const bridge = { getAccessToken: provider, handleUnauthorized: vi.fn() }
    registerAuthBridge(bridge)

    await apiRequest<unknown>('/protected')

    const [, requestInit] = fetchMock.mock.calls[0] ?? []
    const requestHeaders = new Headers((requestInit?.headers as HeadersInit | undefined) ?? {})

    expect(provider).toHaveBeenCalledTimes(1)
    expect(requestHeaders.get('Authorization')).toBe('Bearer access-token')

    clearAuthBridge(bridge)
  })

  it('retries a 401 response once with a newly obtained access token', async () => {
    vi.stubGlobal('fetch', fetchMock)
    fetchMock
      .mockResolvedValueOnce({ ok: false, status: 401, json: async () => ({}) })
      .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({}) })
    const provider = vi.fn().mockResolvedValue('refreshed-access-token')
    const bridge = { getAccessToken: provider, handleUnauthorized: vi.fn() }
    registerAuthBridge(bridge)

    await apiRequest<unknown>('/protected')

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(provider).toHaveBeenCalledTimes(2)
    clearAuthBridge(bridge)
  })
})
