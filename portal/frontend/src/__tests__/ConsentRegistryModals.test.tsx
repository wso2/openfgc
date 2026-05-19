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

import { fireEvent, render, screen } from '@testing-library/react'
import { I18nextProvider } from 'react-i18next'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { OxygenTheme, OxygenUIThemeProvider } from '@wso2/oxygen-ui'
import i18n from '../i18n/i18n'
import ConsentApprovalDialog from '../features/consent-registry/components/ConsentApprovalDialog'
import ConsentRevocationDialog from '../features/consent-registry/components/ConsentRevocationDialog'

function renderWithProviders(component: React.JSX.Element): void {
  render(
    <I18nextProvider i18n={i18n}>
      <OxygenUIThemeProvider theme={OxygenTheme}>{component}</OxygenUIThemeProvider>
    </I18nextProvider>,
  )
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.runOnlyPendingTimers()
  vi.useRealTimers()
})

describe('consent registry dialogs', () => {
  it('shows loading text instead of empty states while approval details load', () => {
    renderWithProviders(
      <ConsentApprovalDialog
        open
        consentId="consent-123"
        loading
        purposes={[]}
        onClose={vi.fn()}
        onConfirm={vi.fn()}
      />,
    )

    expect(screen.getByText('Loading consent details...')).toBeInTheDocument()
    expect(
      screen.queryByText('No mandatory requirements for this consent.'),
    ).not.toBeInTheDocument()
  })

  it('submits selected optional permissions from approval dialog', () => {
    const onConfirm = vi.fn()

    renderWithProviders(
      <ConsentApprovalDialog
        open
        consentId="consent-123"
        loading={false}
        purposes={[
          {
            name: 'Accounts',
            elements: [
              { name: 'Account Number', isUserApproved: true, isMandatory: true },
              { name: 'Transaction History', isUserApproved: true, isMandatory: false },
              { name: 'Marketing Messages', isUserApproved: false, isMandatory: false },
            ],
          },
        ]}
        onClose={vi.fn()}
        onConfirm={onConfirm}
      />,
    )

    const toggles = screen.getAllByRole('checkbox', { name: /toggle permission/i })
    fireEvent.click(toggles[1])

    fireEvent.click(screen.getByRole('button', { name: /approve & continue/i }))

    expect(onConfirm).toHaveBeenCalledWith([
      { purposeName: 'Accounts', elementName: 'Transaction History' },
      { purposeName: 'Accounts', elementName: 'Marketing Messages' },
    ])
  })

  it('calls revocation handlers from confirmation dialog', () => {
    const onConfirm = vi.fn()
    const onClose = vi.fn()

    renderWithProviders(
      <ConsentRevocationDialog
        open
        consentId="consent-456"
        loading={false}
        onClose={onClose}
        onConfirm={onConfirm}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /revoke consents/i }))
    expect(onConfirm).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
