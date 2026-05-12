import { fireEvent, render, screen } from '@testing-library/react'
import { I18nextProvider } from 'react-i18next'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { AcrylicOrangeTheme, CssBaseline, OxygenUIThemeProvider } from '@wso2/oxygen-ui'
import AppSidebar from '../components/layout/sidebar/AppSidebar'
import i18n from '../i18n/i18n'

function LocationProbe(): React.JSX.Element {
  const location = useLocation()

  return <p>{`${location.pathname}${location.search}`}</p>
}

describe('AppSidebar', () => {
  it('renders navigation items and navigates to dashboard and pending consents', () => {
    render(
      <OxygenUIThemeProvider theme={AcrylicOrangeTheme}>
        <CssBaseline />
        <I18nextProvider i18n={i18n}>
          <MemoryRouter initialEntries={['/consents']}>
            <Routes>
              <Route
                path="*"
                element={
                  <>
                    <AppSidebar collapsed={false} />
                    <LocationProbe />
                  </>
                }
              />
            </Routes>
          </MemoryRouter>
        </I18nextProvider>
      </OxygenUIThemeProvider>,
    )

    expect(screen.getByRole('complementary')).toBeInTheDocument()
    expect(screen.getByRole('navigation')).toBeInTheDocument()
    expect(screen.getByText('Consent')).toBeInTheDocument()
    expect(screen.getByText('/consents')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Pending Consents'))

    expect(screen.getByText('/consents?status=Pending')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Dashboard'))

    expect(screen.getByText('/dashboard')).toBeInTheDocument()
  })
})
