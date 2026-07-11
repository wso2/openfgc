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

import { ThunderIDProvider, useThunderID } from '@thunderid/react'
import { useEffect, useMemo, useState } from 'react'
import type { PropsWithChildren } from 'react'
import { clearAuthBridge, registerAuthBridge } from '../authBridge'
import { PortalAuthContext } from '../PortalAuthContext'
import type { PortalAuth } from '../PortalAuthContext'

interface ThunderAuthConfiguration {
  afterSignInUrl: string
  afterSignOutUrl: string
  baseUrl: string
  clientId: string
  scopes: string[]
  tokenRequest?: {
    params: Record<string, string>
  }
}

function getThunderAuthConfiguration(): ThunderAuthConfiguration | undefined {
  const baseUrl = import.meta.env.VITE_THUNDERID_BASE_URL?.trim()
  const clientId = import.meta.env.VITE_THUNDERID_CLIENT_ID?.trim()

  if (!baseUrl || !clientId) {
    return undefined
  }

  const scopes = (import.meta.env.VITE_AUTH_SCOPES ?? 'openid profile email')
    .split(/\s+/)
    .map((scope: string) => scope.trim())
    .filter(Boolean)
  const resource = import.meta.env.VITE_AUTH_RESOURCE?.trim()

  return {
    afterSignInUrl: window.location.origin,
    afterSignOutUrl: window.location.origin,
    baseUrl,
    clientId,
    scopes,
    ...(resource
      ? {
          tokenRequest: {
            params: {
              resource,
            },
          },
        }
      : {}),
  }
}

function ThunderAuthBridge({ children }: PropsWithChildren): React.JSX.Element {
  const thunder = useThunderID()
  const [idTokenUser, setIdTokenUser] = useState<unknown>()
  const { getAccessToken, getDecodedIdToken, isSignedIn, user: sdkUser } = thunder
  const user = isSignedIn ? (idTokenUser ?? sdkUser) : sdkUser

  useEffect(() => {
    const bridge = {
      getAccessToken: async (): Promise<string | undefined> => {
        const accessToken = await getAccessToken()
        return accessToken.trim() || undefined
      },
      handleUnauthorized: thunder.signIn,
    }
    registerAuthBridge(bridge)

    return () => {
      clearAuthBridge(bridge)
    }
  }, [getAccessToken, thunder.signIn])

  useEffect(() => {
    let active = true

    if (!isSignedIn) {
      return undefined
    }

    getDecodedIdToken()
      .then((idToken) => {
        if (!active) {
          return
        }

        const userClaims = sdkUser && typeof sdkUser === 'object' ? sdkUser : {}
        setIdTokenUser({ ...userClaims, ...idToken })
      })
      .catch(() => undefined)

    return () => {
      active = false
    }
  }, [getDecodedIdToken, isSignedIn, sdkUser])

  const value = useMemo<PortalAuth>(
    () => ({
      isAuthenticated: thunder.isSignedIn,
      isInitialized: thunder.isInitialized,
      isLoading: thunder.isLoading,
      getAccessToken: async (): Promise<string | undefined> => {
        const accessToken = await getAccessToken()
        return accessToken.trim() || undefined
      },
      signIn: thunder.signIn,
      signOut: async (): Promise<unknown> => {
        setIdTokenUser(undefined)
        return thunder.signOut()
      },
      user,
    }),
    [thunder, getAccessToken, user],
  )

  return <PortalAuthContext.Provider value={value}>{children}</PortalAuthContext.Provider>
}

function AuthConfigurationRequired(): React.JSX.Element {
  return (
    <main role="alert">
      ThunderID authentication requires `VITE_THUNDERID_BASE_URL` and `VITE_THUNDERID_CLIENT_ID`.
    </main>
  )
}

export function ThunderAuthProvider({ children }: PropsWithChildren): React.JSX.Element {
  const configuration = getThunderAuthConfiguration()

  if (!configuration) {
    return <AuthConfigurationRequired />
  }

  return (
    <ThunderIDProvider
      afterSignInUrl={configuration.afterSignInUrl}
      afterSignOutUrl={configuration.afterSignOutUrl}
      baseUrl={configuration.baseUrl}
      clientId={configuration.clientId}
      scopes={configuration.scopes}
      tokenRequest={configuration.tokenRequest}
    >
      <ThunderAuthBridge>{children}</ThunderAuthBridge>
    </ThunderIDProvider>
  )
}
