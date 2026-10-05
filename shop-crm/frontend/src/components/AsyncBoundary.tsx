import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

interface Props {
  isLoading: boolean
  error: string | null
  onRetry?: () => void
  children: ReactNode
}

/** Wraps a page body so loading and error states are never forgotten. */
export default function AsyncBoundary({ isLoading, error, onRetry, children }: Props) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <div role="status" aria-live="polite" className="py-10 text-center text-sm text-slate-500">
        {t('common.loading')}
      </div>
    )
  }

  if (error) {
    return (
      <div role="alert" className="card mx-auto mt-6 max-w-md p-4 text-center">
        <p className="text-sm font-semibold text-brand-red">{t('common.error')}</p>
        <p className="mt-1 text-sm text-slate-600">{error}</p>
        {onRetry && (
          <button type="button" className="btn-secondary mt-3" onClick={onRetry}>
            {t('common.retry')}
          </button>
        )}
      </div>
    )
  }

  return <>{children}</>
}