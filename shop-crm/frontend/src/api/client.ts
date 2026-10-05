import type { Customer, Product, Sale, SalesPage, ShopConfig } from './types'

/** Error carrying the server's machine-readable code so the UI can translate it. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly params: Record<string, string>
  /** English text from the server, used when no translation exists for the code. */
  readonly fallbackMessage: string

  constructor(status: number, code: string, message: string, params: Record<string, string> = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.params = params
    this.fallbackMessage = message
  }
}

const BASE = '/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await fetch(`${BASE}${path}`, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
    })
  } catch {
    throw new ApiError(0, 'NETWORK_ERROR', 'network')
  }

  if (response.status === 204) {
    return undefined as T
  }

  const payload = await response.json().catch(() => null)

  if (!response.ok) {
    const error = (payload as { error?: { code?: string; message?: string; params?: Record<string, string> } })?.error
    throw new ApiError(
      response.status,
      error?.code ?? 'UNKNOWN_ERROR',
      error?.message ?? response.statusText,
      error?.params ?? {},
    )
  }

  return payload as T
}

export const api = {
  config: () => request<ShopConfig>('/config'),

  listProducts: () => request<Product[]>('/products'),

  createProduct: (body: { name: string; price: string; initialQty: number }) =>
    request<Product>('/products', { method: 'POST', body: JSON.stringify(body) }),

  restock: (id: number, qty: number) =>
    request<Product>(`/products/${id}/restock`, { method: 'POST', body: JSON.stringify({ qty }) }),

  deleteProduct: (id: number) => request<void>(`/products/${id}`, { method: 'DELETE' }),

  listCustomers: (search: string) =>
    request<Customer[]>(`/customers${search ? `?search=${encodeURIComponent(search)}` : ''}`),

  createCustomer: (body: { name: string; phone: string }) =>
    request<Customer>('/customers', { method: 'POST', body: JSON.stringify(body) }),

  listSales: (page: number) => request<SalesPage>(`/sales?page=${page}`),

  createSale: (body: { customerId: number | null; bonusPercent: number; items: { productId: number; qty: number }[] }) =>
    request<Sale>('/sales', { method: 'POST', body: JSON.stringify(body) }),
}