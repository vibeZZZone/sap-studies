import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import en from './locales/en.json'
import ru from './locales/ru.json'

export const LANGUAGES = [
  { code: 'ru', label: 'Русский' },
  { code: 'en', label: 'English' },
] as const

export type LanguageCode = (typeof LANGUAGES)[number]['code']

const STORAGE_KEY = 'shop-crm-language'

function initialLanguage(): LanguageCode {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored && LANGUAGES.some((l) => l.code === stored)) {
    return stored as LanguageCode
  }
  return 'ru'
}

void i18n.use(initReactI18next).init({
  resources: {
    ru: { translation: ru },
    en: { translation: en },
  },
  lng: initialLanguage(),
  fallbackLng: 'ru',
  interpolation: { escapeValue: false },
})

export function setLanguage(code: LanguageCode): void {
  localStorage.setItem(STORAGE_KEY, code)
  void i18n.changeLanguage(code)
  document.documentElement.lang = code
}

export default i18n