import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { apiRequest } from '../api/client'
import type {
  AuthResponse,
  LoginRequest,
  OutletSummary,
  PinLoginRequest,
  RegisterRequest,
  TenantSummary,
  UserSummary,
} from '../types/auth'

const STORAGE_KEY = 'umkm_pos_auth_session'

interface StoredSession {
  accessToken: string
  refreshToken: string
  expiresAt: string
  user: UserSummary
  tenant: TenantSummary
  permissions: string[]
  outlets: OutletSummary[]
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const expiresAt = ref<string | null>(null)
  const user = ref<UserSummary | null>(null)
  const tenant = ref<TenantSummary | null>(null)
  const permissions = ref<string[]>([])
  const outlets = ref<OutletSummary[]>([])

  function hydrate() {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return
    try {
      const parsed = JSON.parse(raw) as StoredSession
      accessToken.value = parsed.accessToken
      refreshToken.value = parsed.refreshToken
      expiresAt.value = parsed.expiresAt
      user.value = parsed.user
      tenant.value = parsed.tenant
      permissions.value = parsed.permissions || []
      outlets.value = parsed.outlets || []
    } catch {
      localStorage.removeItem(STORAGE_KEY)
    }
  }

  hydrate()

  const isAuthenticated = computed(() => Boolean(accessToken.value && user.value))

  function applySession(data: AuthResponse) {
    accessToken.value = data.token.access_token
    refreshToken.value = data.token.refresh_token
    expiresAt.value = data.token.expires_at
    user.value = data.user
    tenant.value = data.tenant
    permissions.value = data.permissions || []
    outlets.value = data.outlets || []

    const snapshot: StoredSession = {
      accessToken: data.token.access_token,
      refreshToken: data.token.refresh_token,
      expiresAt: data.token.expires_at,
      user: data.user,
      tenant: data.tenant,
      permissions: permissions.value,
      outlets: outlets.value,
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot))
  }

  async function login(payload: LoginRequest): Promise<AuthResponse> {
    const data = await apiRequest<AuthResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
    applySession(data)
    return data
  }

  async function pinLogin(payload: PinLoginRequest): Promise<AuthResponse> {
    const data = await apiRequest<AuthResponse>('/api/v1/auth/login/pin', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
    applySession(data)
    return data
  }

  async function register(payload: RegisterRequest): Promise<AuthResponse> {
    const data = await apiRequest<AuthResponse>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
    applySession(data)
    return data
  }

  async function fetchMe(): Promise<void> {
    if (!accessToken.value) return
    const data = await apiRequest<{
      user_id: number
      user_uuid: string
      tenant_id: number
      username: string
      role: string
      permissions: string[]
      outlet_ids: number[]
    }>('/api/v1/auth/me', { method: 'GET' }, accessToken.value)

    if (data.permissions) {
      permissions.value = data.permissions
    }
  }

  function logout() {
    accessToken.value = null
    refreshToken.value = null
    expiresAt.value = null
    user.value = null
    tenant.value = null
    permissions.value = []
    outlets.value = []
    localStorage.removeItem(STORAGE_KEY)
  }

  return {
    accessToken,
    refreshToken,
    expiresAt,
    user,
    tenant,
    permissions,
    outlets,
    isAuthenticated,
    login,
    pinLogin,
    register,
    fetchMe,
    logout,
  }
})
