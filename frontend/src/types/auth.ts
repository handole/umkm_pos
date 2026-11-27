export interface ApiEnvelope<T> {
  success: boolean
  code: number
  message: string
  data?: T
  error?: unknown
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_at: string
}

export interface UserSummary {
  id: number
  uuid: string
  tenant_id: number
  username: string
  full_name: string
  email?: string
  phone?: string
  role: string
  is_active: boolean
  created_at: string
}

export interface TenantSummary {
  id: number
  uuid: string
  code: string
  name: string
}

export interface OutletSummary {
  id: number
  code: string
  name: string
  is_main_branch: boolean
}

export interface AuthResponse {
  token: TokenPair
  user: UserSummary
  tenant: TenantSummary
  permissions: string[]
  outlets: OutletSummary[]
}

export interface LoginRequest {
  username: string
  password: string
  tenant_code?: string
}

export interface PinLoginRequest {
  username: string
  pin: string
  outlet_id?: number
}

export type RoleName = 'SUPERADMIN' | 'STORE_MANAGER' | 'CASHIER' | 'WAREHOUSE' | 'HR_FINANCE'

export interface RegisterRequest {
  username: string
  email?: string
  phone?: string
  password: string
  full_name: string
  role_name: RoleName
  pin?: string
  tenant_code?: string
  tenant_name?: string
  outlet_ids?: number[]
}
