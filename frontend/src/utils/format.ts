export function formatIDR(val: number | undefined | null): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(val || 0)
}

export function formatDateTime(iso: string | undefined | null): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

export function formatDate(iso: string | undefined | null): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleDateString('id-ID', {
    dateStyle: 'medium',
  })
}
