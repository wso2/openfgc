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
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import { AsgardeoProvider, useAsgardeo } from '@asgardeo/react'
import { useEffect, useMemo } from 'react'
import type { PropsWithChildren } from 'react'
import { clearAuthBridge, registerAuthBridge } from '../authBridge'
import { PortalAuthContext } from '../PortalAuthContext'

interface AsgardeoAuthConfiguration {
  afterSignInUrl: string
  afterSignOutUrl: string
  baseUrl: string
  clientId: string
  scopes: string[]
  signInOptions?: {
    resource: string
  }
}

function getAsgardeoAuthConfiguration(): AsgardeoAuthConfiguration | undefined {
  const baseUrl = import.meta.env.VITE_ASGARDEO_BASE_URL?.trim()
  const clientId = import.meta.env.VITE_ASGARDEO_CLIENT_ID?.trim()

  if (!baseUrl || !clientId) {
    return undefined
  }

  const resource = import.meta.env.VITE_AUTH_RESOURCE?.trim()

  return {
    afterSignInUrl: window.location.origin,
    afterSignOutUrl: window.location.origin,
    baseUrl,
    clientId,
    scopes: (import.meta.env.VITE_AUTH_SCOPES ?? 'openid profile email')
      .split(/\s+/)
      .map((scope: string) => scope.trim())
      .filter(Boolean),
    ...(resource ? { signInOptions: { resource } } : {}),
  }
}

function AsgardeoAuthBridge({ children }: PropsWithChildren): React.JSX.Element {
  const asgardeo = useAsgardeo()
  const { getAccessToken, signIn } = asgardeo

  useEffect(() => {
    const bridge = {
      getAccessToken: async (): Promise<string | undefined> => {
        const accessToken = await getAccessToken()
        return accessToken.trim() || undefined
      },
      handleUnauthorized: signIn,
    }
    registerAuthBridge(bridge)

    return () => {
      clearAuthBridge(bridge)
    }
  }, [getAccessToken, signIn])

  const value = useMemo(
    () => ({
      isAuthenticated: asgardeo.isSignedIn,
      isInitialized: asgardeo.isInitialized,
      isLoading: asgardeo.isLoading,
      getAccessToken: async (): Promise<string | undefined> => {
        const accessToken = await asgardeo.getAccessToken()
        return accessToken.trim() || undefined
      },
      signIn: asgardeo.signIn,
      signOut: asgardeo.signOut,
      user: asgardeo.user,
    }),
    [asgardeo],
  )

  return <PortalAuthContext.Provider value={value}>{children}</PortalAuthContext.Provider>
}

function AuthConfigurationRequired(): React.JSX.Element {
  return (
    <main role="alert">
      Asgardeo authentication requires `VITE_ASGARDEO_BASE_URL` and `VITE_ASGARDEO_CLIENT_ID`.
    </main>
  )
}

export function AsgardeoAuthProvider({ children }: PropsWithChildren): React.JSX.Element {
  const configuration = getAsgardeoAuthConfiguration()

  if (!configuration) {
    return <AuthConfigurationRequired />
  }

  return (
    <AsgardeoProvider
      afterSignInUrl={configuration.afterSignInUrl}
      afterSignOutUrl={configuration.afterSignOutUrl}
      baseUrl={configuration.baseUrl}
      clientId={configuration.clientId}
      scopes={configuration.scopes}
      signInOptions={configuration.signInOptions}
    >
      <AsgardeoAuthBridge>{children}</AsgardeoAuthBridge>
    </AsgardeoProvider>
  )
}
