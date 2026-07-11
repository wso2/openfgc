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

export interface AuthBridge {
  getAccessToken: () => Promise<string | undefined>
  handleUnauthorized: () => Promise<unknown>
}

let activeBridge: AuthBridge | undefined
let unauthorizedHandled = false

export function registerAuthBridge(bridge: AuthBridge): void {
  activeBridge = bridge
  unauthorizedHandled = false
}

export function clearAuthBridge(bridge: AuthBridge): void {
  if (activeBridge === bridge) {
    activeBridge = undefined
  }
}

export async function getAccessToken(): Promise<string | undefined> {
  return activeBridge?.getAccessToken()
}

export async function handleUnauthorized(): Promise<void> {
  if (!activeBridge || unauthorizedHandled) {
    return
  }

  unauthorizedHandled = true
  await activeBridge.handleUnauthorized()
}
