export interface Product {
  id: number
  name: string
  priceKop: number
  price: number
  stock: number
  archived: boolean
  lowStock: boolean
  createdAt: string
}

export interface Customer {
  id: number
  cardNumber: string
  name: string
  phone: string
  bonusBalance: number
  totalSpentKop: number
  totalSpent: string
  createdAt: string
}

export interface SaleItem {
  productId: number
  name: string
  priceKop: number
  price: string
  qty: number
  sumKop: number
  sum: string
}

export interface Sale {
  id: number
  customerId: number | null
  customer: { id: number; name: string; cardNumber: string } | null
  subtotalKop: number
  subtotal: string
  bonusSpent: number
  totalKop: number
  total: string
  bonusEarned: number
  createdAt: string
  items: SaleItem[]
}

export interface SalesPage {
  sales: Sale[]
  page: number
  pageSize: number
  total: number
  pages: number
}

export interface ShopConfig {
  cashbackPercent: number
  bonusSpendLimitPct: number
  kopPerRuble: number
  currency: string
}