import type { ApiEnvelope } from '../types/auth'

export class ApiError extends Error {
  status: number
  details?: unknown

  constructor(message: string, status: number, details?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.details = details
  }
}

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
  token?: string | null,
): Promise<T> {
  const headers = new Headers(options.headers)
  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  let payload: ApiEnvelope<T> | null = null
  try {
    payload = (await res.json()) as ApiEnvelope<T>
  } catch {
    throw new ApiError('Server tidak mengembalikan respons JSON yang valid.', res.status)
  }

  if (!res.ok || !payload.success) {
    throw new ApiError(
      payload.message || 'Permintaan gagal diproses.',
      res.status,
      payload.error,
    )
  }

  return payload.data as T
}

export function unwrapList<T>(data: T[] | { items?: T[] } | null | undefined): T[] {
  if (!data) return []
  if (Array.isArray(data)) return data
  return Array.isArray(data.items) ? data.items : []
}
