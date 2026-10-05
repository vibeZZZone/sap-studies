import { useState } from 'react'
import { NavLink, Navigate, Route, Routes } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

import { LANGUAGES, setLanguage, type LanguageCode } from './i18n'
import CustomersPage from './pages/CustomersPage'
import ProductsPage from './pages/ProductsPage'
import SalesPage from './pages/SalesPage'
import TillPage from './pages/TillPage'

const NAV = [
  { to: '/till', key: 'till' },
  { to: '/products', key: 'products' },
  { to: '/customers', key: 'customers' },
  { to: '/sales', key: 'sales' },
] as const

export default function App() {
  const { t, i18n } = useTranslation()
  const [language, setCurrent] = useState<LanguageCode>(i18n.language as LanguageCode)

  function changeLanguage(code: LanguageCode): void {
    setLanguage(code)
    setCurrent(code)
  }

  return (
    <div className="min-h-dvh">
      <header className="sticky top-0 z-10 border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-4 gap-y-2 px-3 py-3 sm:px-4">
          <span className="text-lg font-bold text-brand-red">{t('app.title')}</span>

          <nav aria-label={t('app.title')} className="order-3 flex w-full gap-1 overflow-x-auto sm:order-none sm:w-auto">
            {NAV.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  [
                    'whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium transition',
                    isActive ? 'bg-brand-red text-white' : 'text-slate-600 hover:bg-slate-100',
                  ].join(' ')
                }
              >
                {t(`app.nav.${item.key}`)}
              </NavLink>
            ))}
          </nav>

          <div className="ms-auto flex items-center gap-2">
            <label htmlFor="language" className="text-xs text-slate-500">
              {t('app.language')}
            </label>
            <select
              id="language"
              className="field h-10 min-h-[40px] w-auto py-1 text-sm"
              value={language}
              onChange={(event) => changeLanguage(event.target.value as LanguageCode)}
            >
              {LANGUAGES.map((lang) => (
                <option key={lang.code} value={lang.code}>
                  {lang.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-3 py-4 sm:px-4 sm:py-6">
        <Routes>
          <Route path="/" element={<Navigate to="/till" replace />} />
          <Route path="/till" element={<TillPage />} />
          <Route path="/products" element={<ProductsPage />} />
          <Route path="/customers" element={<CustomersPage />} />
          <Route path="/sales" element={<SalesPage />} />
          <Route path="*" element={<Navigate to="/till" replace />} />
        </Routes>
      </main>
    </div>
  )
}