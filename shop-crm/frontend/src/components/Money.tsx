import { useTranslation } from 'react-i18next'

import { formatBonuses, formatKop } from '../lib/money'

interface Props {
  kop: number
  bonus?: number
  className?: string
}

/** Renders a money amount the way the till shows it: 1 234 ₽. */
export default function Money({ kop, bonus, className }: Props) {
  const { t } = useTranslation()
  return (
    <span className={className} title={t('customers.bonusHint')}>
      {formatKop(kop)}
      {bonus !== undefined && bonus > 0 && (
        <span className="ms-2 text-brand-green">+{formatBonuses(bonus)}</span>
      )}
    </span>
  )
}