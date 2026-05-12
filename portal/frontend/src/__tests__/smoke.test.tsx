import { render, screen } from '@testing-library/react'
import { I18nextProvider, useTranslation } from 'react-i18next'
import { describe, expect, it } from 'vitest'
import i18n from '../i18n/i18n'

function I18nPlaceholder(): React.JSX.Element {
  const { t } = useTranslation('common')

  return <h1>{t('app.title')}</h1>
}

describe('app i18n setup', () => {
  it('renders translated heading text with i18n provider', () => {
    render(
      <I18nextProvider i18n={i18n}>
        <I18nPlaceholder />
      </I18nextProvider>,
    )

    expect(
      screen.getByRole('heading', {
        name: String(i18n.t('common:app.title')),
      }),
    ).toBeInTheDocument()
  })
})
