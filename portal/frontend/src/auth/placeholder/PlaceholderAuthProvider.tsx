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

import { useEffect, useMemo } from 'react'
import type { PropsWithChildren } from 'react'
import { clearAuthBridge, registerAuthBridge } from '../authBridge'
import { PortalAuthContext } from '../PortalAuthContext'
import type { PortalAuth } from '../PortalAuthContext'

function LocalPlaceholderAuthProvider({ children }: PropsWithChildren): React.JSX.Element {
  const auth = useMemo<PortalAuth>(
    () => ({
      getAccessToken: async (): Promise<string | undefined> => undefined,
      isAuthenticated: true,
      isInitialized: true,
      isLoading: false,
      signIn: async (): Promise<void> => undefined,
      signOut: async (): Promise<void> => undefined,
      user: {
        name: 'Local development',
      },
    }),
    [],
  )

  useEffect(() => {
    const bridge = {
      getAccessToken: auth.getAccessToken,
      handleUnauthorized: auth.signIn,
    }
    registerAuthBridge(bridge)

    return () => {
      clearAuthBridge(bridge)
    }
  }, [auth])

  return <PortalAuthContext.Provider value={auth}>{children}</PortalAuthContext.Provider>
}

export default function PlaceholderAuthProvider({
  children,
}: PropsWithChildren): React.JSX.Element {
  if (import.meta.env.PROD) {
    return (
      <main role="alert">Placeholder authentication is available only in local development.</main>
    )
  }

  return <LocalPlaceholderAuthProvider>{children}</LocalPlaceholderAuthProvider>
}
