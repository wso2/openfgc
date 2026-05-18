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

import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import MainLayout from './components/layout/main-layout/MainLayout'
import ConsentDetailsPage from './features/consent-registry/ConsentDetailsPage'
import ConsentRegistryPage from './features/consent-registry/ConsentRegistryPage'
import DashboardPage from './features/dashboard/DashboardPage'

function ConsentRegistryRoute(): React.JSX.Element {
  const location = useLocation()

  return <ConsentRegistryPage key={location.search} />
}

function App(): React.JSX.Element {
  return (
    <Routes>
      <Route element={<MainLayout />}>
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/consents" element={<ConsentRegistryRoute />} />
        <Route path="/consents/:id" element={<ConsentDetailsPage />} />
        <Route path="*" element={<Navigate to="/consents" replace />} />
      </Route>
    </Routes>
  )
}

export default App
