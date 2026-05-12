import { render, screen } from '@testing-library/react'
import { AcrylicOrangeTheme, CssBaseline, OxygenUIThemeProvider } from '@wso2/oxygen-ui'
import { I18nextProvider } from 'react-i18next'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import ConsentRegistryPage from '../features/consent-registry/ConsentRegistryPage'
import i18n from '../i18n/i18n'

describe('ConsentRegistryPage', () => {
  it('renders page heading, filters, and grouped consent rows', () => {
    render(
      <OxygenUIThemeProvider theme={AcrylicOrangeTheme}>
        <CssBaseline />
        <I18nextProvider i18n={i18n}>
          <MemoryRouter>
            <ConsentRegistryPage />
          </MemoryRouter>
        </I18nextProvider>
      </OxygenUIThemeProvider>,
    )

    expect(screen.getByRole('heading', { name: 'All Consents' })).toBeInTheDocument()
    expect(screen.getByLabelText('Consent filters')).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'Consent registry table' })).toBeInTheDocument()
    expect(screen.getByText('Client: Tesco_Bank_v1')).toBeInTheDocument()
    expect(screen.getByText('#CON-8291')).toBeInTheDocument()
  })
})
