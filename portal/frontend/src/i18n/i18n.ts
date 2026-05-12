import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import commonEn from './resources/en/common'

const resources = {
  en: {
    common: commonEn,
  },
} as const

i18n.use(initReactI18next).init({
  resources,
  lng: 'en',
  fallbackLng: 'en',
  defaultNS: 'common',
  ns: ['common'],
  interpolation: {
    escapeValue: false,
  },
})

export default i18n
