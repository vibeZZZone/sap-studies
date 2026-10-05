import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from './client'
import type { Customer, Product, Sale, SalesPage, ShopConfig } from './types'

export const queryKeys = {
  config: ['config'] as const,
  products: ['products'] as const,
  customers: (search: string) => ['customers', search] as const,
  sales: (page: number) => ['sales', page] as const,
}

export function useShopConfig() {
  return useQuery<ShopConfig>({ queryKey: queryKeys.config, queryFn: api.config, staleTime: Infinity })
}

export function useProducts() {
  return useQuery<Product[]>({ queryKey: queryKeys.products, queryFn: api.listProducts })
}

export function useCreateProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createProduct,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.products }),
  })
}

export function useRestockProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, qty }: { id: number; qty: number }) => api.restock(id, qty),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.products }),
  })
}

export function useDeleteProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteProduct,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.products }),
  })
}

// Debounced so typing in the till picker does not fire a request per keystroke.
export function useCustomers(search: string, options?: { enabled?: boolean }) {
  const debounced = useDebouncedValue(search, 250)
  return useQuery<Customer[]>({
    queryKey: queryKeys.customers(debounced),
    queryFn: () => api.listCustomers(debounced),
    enabled: options?.enabled ?? true,
  })
}

function useDebouncedValue<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(timer)
  }, [value, delay])
  return debounced
}

export function useCreateCustomer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createCustomer,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['customers'] }),
  })
}

export function useSales(page: number) {
  return useQuery<SalesPage>({
    queryKey: queryKeys.sales(page),
    queryFn: () => api.listSales(page),
    // Refreshing the history after a sale would be pointless; it changes only when the
    // till posts a new receipt.
    staleTime: 5_000,
  })
}

export function useCreateSale() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createSale,
    onSuccess: (sale: Sale) => {
      qc.invalidateQueries({ queryKey: ['sales'] })
      qc.invalidateQueries({ queryKey: queryKeys.products })
      qc.invalidateQueries({ queryKey: ['customers'] })
      void sale
    },
  })
}