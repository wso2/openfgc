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

import type { PropsWithChildren } from 'react'
import { AsgardeoAuthProvider } from './asgardeo/AsgardeoAuthProvider'
import PlaceholderAuthProvider from './placeholder/PlaceholderAuthProvider'
import { ThunderAuthProvider } from './thunder/ThunderAuthProvider'

export function PortalAuthProvider({ children }: PropsWithChildren): React.JSX.Element {
  const provider = (import.meta.env.VITE_AUTH_PROVIDER ?? 'thunder').trim().toLowerCase()

  if (provider === 'asgardeo') {
    return <AsgardeoAuthProvider>{children}</AsgardeoAuthProvider>
  }

  if (provider === 'placeholder') {
    return <PlaceholderAuthProvider>{children}</PlaceholderAuthProvider>
  }

  if (provider === 'thunder') {
    return <ThunderAuthProvider>{children}</ThunderAuthProvider>
  }

  return <main role="alert">Unsupported `VITE_AUTH_PROVIDER`: {provider}</main>
}
