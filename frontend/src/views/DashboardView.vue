<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError, apiRequest, unwrapList } from '../api/client'
import AppShell from '../components/AppShell.vue'
import { useAuthStore } from '../stores/auth'

interface PaymentBreakdownItem {
  method_code: string
  total: number
  count: number
}

interface SalesReportSummary {
  start_date: string
  end_date: string
  total_orders: number
  completed_orders: number
  void_orders: number
  gross_sales: number
  total_discount: number
  total_tax: number
  net_sales: number
  total_cogs: number
  gross_profit: number
  payment_breakdown?: PaymentBreakdownItem[]
}

interface PosShift {
  uuid: string
  outlet_name?: string
  status: string
  starting_cash: number
  expected_cash: number
  opened_at: string
}

interface PosOrderItemSummary {
  uuid: string
  order_number: string
  status: string
  total_amount: number
  ordered_at: string
}

interface StockItem {
  uuid: string
  location_name: string
  variant_sku: string
  variant_name: string
  quantity: number
  min_stock_alert: number
}

const auth = useAuthStore()

function hasPerm(perm: string): boolean {
  if (auth.user?.role === 'SUPERADMIN') return true
  return auth.permissions.includes(perm)
}

const canReadReport = computed(() => hasPerm('payroll:read'))
const canReadShift = computed(() => hasPerm('pos:shift'))
const canReadOrders = computed(() => hasPerm('pos:checkout'))
const canReadInventory = computed(() => hasPerm('inventory:read'))

const dateFilter = reactive({
  start_date: '',
  end_date: '',
})

const isRefreshing = ref(false)

const reportState = reactive<{
  loading: boolean
  error: string
  data: SalesReportSummary | null
}>({ loading: false, error: '', data: null })

const shiftState = reactive<{
  loading: boolean
  error: string
  data: PosShift | null
}>({ loading: false, error: '', data: null })

const ordersState = reactive<{
  loading: boolean
  error: string
  items: PosOrderItemSummary[]
}>({ loading: false, error: '', items: [] })

const stockState = reactive<{
  loading: boolean
  error: string
  items: StockItem[]
}>({ loading: false, error: '', items: [] })

function formatIDR(val: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(val || 0)
}

function formatDateTime(iso: string): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

async function loadReport() {
  if (!canReadReport.value) return
  reportState.loading = true
  reportState.error = ''
  try {
    const params = new URLSearchParams()
    if (dateFilter.start_date) params.set('start_date', dateFilter.start_date)
    if (dateFilter.end_date) params.set('end_date', dateFilter.end_date)
    const qs = params.toString() ? `?${params.toString()}` : ''
    reportState.data = await apiRequest<SalesReportSummary>(
      `/api/v1/pos/reports/summary${qs}`,
      { method: 'GET' },
      auth.accessToken,
    )
  } catch (err) {
    reportState.error = err instanceof ApiError ? err.message : 'Gagal memuat ringkasan penjualan.'
  } finally {
    reportState.loading = false
  }
}

async function loadShift() {
  if (!canReadShift.value) return
  shiftState.loading = true
  shiftState.error = ''
  try {
    shiftState.data = await apiRequest<PosShift>(
      '/api/v1/pos/shifts/current',
      { method: 'GET' },
      auth.accessToken,
    )
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      shiftState.data = null
    } else {
      shiftState.error = err instanceof ApiError ? err.message : 'Gagal memeriksa shift kasir.'
    }
  } finally {
    shiftState.loading = false
  }
}

async function loadRecentOrders() {
  if (!canReadOrders.value) return
  ordersState.loading = true
  ordersState.error = ''
  try {
    const res = await apiRequest<PosOrderItemSummary[] | { items: PosOrderItemSummary[] }>(
      '/api/v1/pos/orders?limit=5&offset=0',
      { method: 'GET' },
      auth.accessToken,
    )
    ordersState.items = unwrapList(res)
  } catch (err) {
    ordersState.error = err instanceof ApiError ? err.message : 'Gagal memuat daftar transaksi.'
  } finally {
    ordersState.loading = false
  }
}

async function loadLowStocks() {
  if (!canReadInventory.value) return
  stockState.loading = true
  stockState.error = ''
  try {
    const res = await apiRequest<StockItem[] | { items: StockItem[] }>(
      '/api/v1/inventory/stocks?low_stock_only=true&limit=5&offset=0',
      { method: 'GET' },
      auth.accessToken,
    )
    stockState.items = unwrapList(res)
  } catch (err) {
    stockState.error = err instanceof ApiError ? err.message : 'Gagal memuat status stok.'
  } finally {
    stockState.loading = false
  }
}

async function loadAll() {
  isRefreshing.value = true
  try {
    await auth.fetchMe()
  } catch {
    // Keep existing local session if /me fails transiently; sub-cards report their own errors
  }
  await Promise.all([loadReport(), loadShift(), loadRecentOrders(), loadLowStocks()])
  isRefreshing.value = false
}

onMounted(() => {
  loadAll()
})
</script>

<template>
  <AppShell
    page-title="Ringkasan Operasional"
    :page-subtitle="auth.tenant ? `${auth.tenant.name} (${auth.tenant.code})` : undefined"
  >
    <template #header-actions>
      <button
        type="button"
        :disabled="isRefreshing"
        class="inline-flex min-h-[44px] items-center rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white hover:bg-[var(--brand-forest-hover)] disabled:opacity-65"
        @click="loadAll"
      >
        {{ isRefreshing ? 'Memuat...' : 'Muat Ulang Data' }}
      </button>
    </template>

    <!-- Top Asymmetric Band: Sales Ledger + Cashier Shift -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
      <!-- Sales Summary Card -->
      <section
        aria-labelledby="sales-summary-heading"
        class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5 lg:col-span-7"
      >
        <div class="flex flex-wrap items-start justify-between gap-3 border-b border-[var(--border-ledger)] pb-4">
          <div>
            <h2 id="sales-summary-heading" class="text-base font-bold text-[var(--ink-primary)]">
              Ringkasan Penjualan
            </h2>
            <p class="mt-0.5 text-xs text-[var(--ink-secondary)]">
              Data langsung dari transaksi kasir yang selesai.
            </p>
          </div>

          <form
            v-if="canReadReport"
            class="flex flex-wrap items-end gap-2"
            @submit.prevent="loadReport"
          >
            <div>
              <label for="report-start" class="block text-xs font-semibold text-[var(--ink-secondary)]">
                Dari
              </label>
              <input
                id="report-start"
                v-model="dateFilter.start_date"
                type="date"
                class="mt-1 min-h-[40px] rounded border border-[var(--border-strong)] bg-[var(--bg-surface)] px-2.5 py-1 text-sm"
              />
            </div>
            <div>
              <label for="report-end" class="block text-xs font-semibold text-[var(--ink-secondary)]">
                Sampai
              </label>
              <input
                id="report-end"
                v-model="dateFilter.end_date"
                type="date"
                class="mt-1 min-h-[40px] rounded border border-[var(--border-strong)] bg-[var(--bg-surface)] px-2.5 py-1 text-sm"
              />
            </div>
            <button
              type="submit"
              class="min-h-[40px] rounded border border-[var(--border-strong)] bg-[var(--bg-subtle)] px-3 py-1.5 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--border-ledger)]"
            >
              Terapkan
            </button>
          </form>
        </div>

        <div v-if="!canReadReport" class="py-6 text-sm text-[var(--ink-secondary)]">
          Peran akun Anda tidak memiliki izin melihat laporan keuangan (<code class="font-mono text-xs">payroll:read</code>).
        </div>
        <div v-else-if="reportState.loading" role="status" class="py-6 text-sm text-[var(--ink-secondary)]">
          Menghitung ringkasan penjualan...
        </div>
        <div
          v-else-if="reportState.error"
          role="alert"
          class="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-3.5 py-3 text-sm text-[var(--error-text)]"
        >
          <span>Error: {{ reportState.error }}</span>
          <button
            type="button"
            class="min-h-[40px] rounded border border-[var(--error-text)] px-3 py-1 text-xs font-semibold"
            @click="loadReport"
          >
            Coba Lagi
          </button>
        </div>
        <div v-else-if="reportState.data" class="mt-4 space-y-4">
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div class="rounded-md bg-[var(--bg-paper)] p-3.5 border border-[var(--border-ledger)]">
              <p class="text-xs font-semibold text-[var(--ink-secondary)]">Omzet Bersih</p>
              <p class="mt-1 text-xl font-bold text-[var(--brand-forest)]">
                {{ formatIDR(reportState.data.net_sales) }}
              </p>
            </div>
            <div class="rounded-md bg-[var(--bg-paper)] p-3.5 border border-[var(--border-ledger)]">
              <p class="text-xs font-semibold text-[var(--ink-secondary)]">Laba Kotor (Est.)</p>
              <p class="mt-1 text-xl font-bold text-[var(--ink-primary)]">
                {{ formatIDR(reportState.data.gross_profit) }}
              </p>
            </div>
            <div class="rounded-md bg-[var(--bg-paper)] p-3.5 border border-[var(--border-ledger)]">
              <p class="text-xs font-semibold text-[var(--ink-secondary)]">Transaksi Selesai</p>
              <p class="mt-1 text-xl font-bold text-[var(--ink-primary)]">
                {{ reportState.data.completed_orders }}
                <span v-if="reportState.data.void_orders > 0" class="text-xs font-normal text-[var(--error-text)]">
                  ({{ reportState.data.void_orders }} batal)
                </span>
              </p>
            </div>
          </div>

          <div v-if="reportState.data.payment_breakdown?.length" class="border-t border-[var(--border-ledger)] pt-3">
            <h3 class="text-xs font-bold uppercase tracking-wide text-[var(--ink-secondary)]">
              Rincian Metode Pembayaran
            </h3>
            <ul class="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
              <li
                v-for="pm in reportState.data.payment_breakdown"
                :key="pm.method_code"
                class="flex items-center justify-between rounded border border-[var(--border-ledger)] px-3 py-2 text-sm"
              >
                <span class="font-semibold">{{ pm.method_code }} ({{ pm.count }}x)</span>
                <span>{{ formatIDR(pm.total) }}</span>
              </li>
            </ul>
          </div>
          <p v-else class="text-sm text-[var(--ink-secondary)]">
            Belum ada transaksi selesai pada rentang tanggal yang dipilih.
          </p>
        </div>
      </section>

      <!-- Active Cashier Shift Card -->
      <section
        aria-labelledby="shift-heading"
        class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5 lg:col-span-5"
      >
        <div class="border-b border-[var(--border-ledger)] pb-4">
          <h2 id="shift-heading" class="text-base font-bold text-[var(--ink-primary)]">
            Status Shift Kasir
          </h2>
          <p class="mt-0.5 text-xs text-[var(--ink-secondary)]">
            Posisi laci kasir aktif milik akun yang sedang masuk.
          </p>
        </div>

        <div v-if="!canReadShift" class="py-6 text-sm text-[var(--ink-secondary)]">
          Akun ini tidak memiliki izin shift kasir (<code class="font-mono text-xs">pos:shift</code>).
        </div>
        <div v-else-if="shiftState.loading" role="status" class="py-6 text-sm text-[var(--ink-secondary)]">
          Memeriksa status shift kasir...
        </div>
        <div
          v-else-if="shiftState.error"
          role="alert"
          class="mt-4 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-3.5 py-3 text-sm text-[var(--error-text)]"
        >
          Error: {{ shiftState.error }}
        </div>
        <div v-else-if="shiftState.data" class="mt-4 space-y-3 text-sm">
          <div class="flex items-center justify-between">
            <span class="text-[var(--ink-secondary)]">Status</span>
            <span class="rounded bg-[var(--brand-forest)] px-2.5 py-0.5 text-xs font-bold text-white">
              {{ shiftState.data.status }}
            </span>
          </div>
          <div class="flex items-center justify-between border-t border-[var(--border-ledger)] pt-2.5">
            <span class="text-[var(--ink-secondary)]">Modal Awal Laci</span>
            <span class="font-semibold">{{ formatIDR(shiftState.data.starting_cash) }}</span>
          </div>
          <div class="flex items-center justify-between border-t border-[var(--border-ledger)] pt-2.5">
            <span class="text-[var(--ink-secondary)]">Perkiraan Kas Saat Ini</span>
            <span class="text-base font-bold text-[var(--brand-forest)]">
              {{ formatIDR(shiftState.data.expected_cash) }}
            </span>
          </div>
          <div class="flex items-center justify-between border-t border-[var(--border-ledger)] pt-2.5">
            <span class="text-[var(--ink-secondary)]">Dibuka Pada</span>
            <span>{{ formatDateTime(shiftState.data.opened_at) }}</span>
          </div>
        </div>
        <div v-else class="py-6 text-sm text-[var(--ink-secondary)]">
          Tidak ada shift kasir yang sedang terbuka untuk akun ini.
        </div>
      </section>
    </div>

    <!-- Middle Band: Responsive Recent Orders Table -->
    <section
      aria-labelledby="orders-table-caption"
      class="mt-6 rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5"
    >
      <div class="mb-4 flex items-center justify-between gap-3">
        <div>
          <h2 id="orders-table-caption" class="text-base font-bold text-[var(--ink-primary)]">
            Transaksi Kasir Terbaru
          </h2>
          <p class="mt-0.5 text-xs text-[var(--ink-secondary)]">
            Maksimal 5 pesanan terakhir dari endpoint POS.
          </p>
        </div>
      </div>

      <div v-if="!canReadOrders" class="py-4 text-sm text-[var(--ink-secondary)]">
        Akun ini tidak memiliki izin membaca transaksi (<code class="font-mono text-xs">pos:checkout</code>).
      </div>
      <div v-else-if="ordersState.loading" role="status" class="py-4 text-sm text-[var(--ink-secondary)]">
        Memuat daftar transaksi...
      </div>
      <div
        v-else-if="ordersState.error"
        role="alert"
        class="rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-3.5 py-3 text-sm text-[var(--error-text)]"
      >
        Error: {{ ordersState.error }}
      </div>
      <div v-else-if="!ordersState.items.length" class="py-4 text-sm text-[var(--ink-secondary)]">
        Belum ada transaksi penjualan yang tercatat.
      </div>
      <div
        v-else
        tabindex="0"
        role="region"
        aria-labelledby="orders-table-caption"
        class="responsive-table-wrap"
      >
        <table class="responsive-table w-full text-left text-sm">
          <caption class="visually-hidden">Transaksi Kasir Terbaru</caption>
          <thead>
            <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
              <th scope="col" class="px-3 py-2.5">Nomor Order</th>
              <th scope="col" class="px-3 py-2.5">Waktu</th>
              <th scope="col" class="px-3 py-2.5">Total</th>
              <th scope="col" class="px-3 py-2.5">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="ord in ordersState.items"
              :key="ord.uuid"
              class="border-b border-[var(--border-ledger)] last:border-b-0"
            >
              <th scope="row" class="px-3 py-3 font-semibold text-[var(--ink-primary)]">
                {{ ord.order_number }}
              </th>
              <td data-label="Waktu" class="px-3 py-3 text-[var(--ink-secondary)]">
                {{ formatDateTime(ord.ordered_at) }}
              </td>
              <td data-label="Total" class="px-3 py-3 font-semibold">
                {{ formatIDR(ord.total_amount) }}
              </td>
              <td data-label="Status" class="px-3 py-3">
                <span
                  class="inline-block rounded px-2 py-0.5 text-xs font-semibold"
                  :class="
                    ord.status === 'VOID'
                      ? 'bg-[var(--error-bg)] text-[var(--error-text)]'
                      : 'bg-[var(--bg-subtle)] text-[var(--brand-forest)]'
                  "
                >
                  {{ ord.status }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Bottom Band: Low Stock Alerts + Outlets & Permissions -->
    <div class="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2">
      <section
        aria-labelledby="low-stock-heading"
        class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5"
      >
        <h2 id="low-stock-heading" class="text-base font-bold text-[var(--ink-primary)]">
          Peringatan Stok Menipis
        </h2>
        <p class="mt-0.5 text-xs text-[var(--ink-secondary)]">
          Varian dengan saldo di bawah batas minimum alert.
        </p>

        <div v-if="!canReadInventory" class="mt-4 text-sm text-[var(--ink-secondary)]">
          Akun ini tidak memiliki izin membaca stok (<code class="font-mono text-xs">inventory:read</code>).
        </div>
        <div v-else-if="stockState.loading" role="status" class="mt-4 text-sm text-[var(--ink-secondary)]">
          Memeriksa saldo stok...
        </div>
        <div
          v-else-if="stockState.error"
          role="alert"
          class="mt-4 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-3.5 py-3 text-sm text-[var(--error-text)]"
        >
          Error: {{ stockState.error }}
        </div>
        <p v-else-if="!stockState.items.length" class="mt-4 text-sm text-[var(--ink-secondary)]">
          Seluruh stok saat ini berada di atas batas minimum.
        </p>
        <ul v-else class="mt-4 divide-y divide-[var(--border-ledger)] text-sm">
          <li
            v-for="item in stockState.items"
            :key="item.uuid"
            class="flex items-center justify-between gap-3 py-2.5"
          >
            <div>
              <p class="font-semibold text-[var(--ink-primary)]">
                {{ item.variant_name }} ({{ item.variant_sku }})
              </p>
              <p class="text-xs text-[var(--ink-secondary)]">{{ item.location_name }}</p>
            </div>
            <span class="font-bold text-[var(--accent-terracotta)]">
              Sisa {{ item.quantity }} (Min {{ item.min_stock_alert }})
            </span>
          </li>
        </ul>
      </section>

      <section
        aria-labelledby="access-heading"
        class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5"
      >
        <h2 id="access-heading" class="text-base font-bold text-[var(--ink-primary)]">
          Outlet &amp; Hak Akses Akun
        </h2>
        <p class="mt-0.5 text-xs text-[var(--ink-secondary)]">
          Cakupan cabang dan izin modul yang aktif pada sesi Anda.
        </p>

        <div class="mt-4">
          <h3 class="text-xs font-bold uppercase tracking-wide text-[var(--ink-secondary)]">Outlet</h3>
          <p v-if="!auth.outlets.length" class="mt-1 text-sm text-[var(--ink-secondary)]">
            Belum ada outlet terhubung pada akun ini.
          </p>
          <ul v-else class="mt-1.5 space-y-1 text-sm">
            <li v-for="o in auth.outlets" :key="o.id" class="flex items-center justify-between">
              <span>{{ o.name }} ({{ o.code }})</span>
              <span v-if="o.is_main_branch" class="text-xs font-semibold text-[var(--brand-forest)]">Utama</span>
            </li>
          </ul>
        </div>

        <div class="mt-4 border-t border-[var(--border-ledger)] pt-3">
          <h3 class="text-xs font-bold uppercase tracking-wide text-[var(--ink-secondary)]">
            Permission ({{ auth.permissions.length }})
          </h3>
          <p v-if="!auth.permissions.length" class="mt-1 text-sm text-[var(--ink-secondary)]">
            Tidak ada permission tercatat.
          </p>
          <ul v-else class="mt-2 flex flex-wrap gap-1.5">
            <li
              v-for="perm in auth.permissions"
              :key="perm"
              class="rounded border border-[var(--border-ledger)] bg-[var(--bg-paper)] px-2 py-0.5 text-xs font-medium text-[var(--ink-primary)]"
            >
              {{ perm }}
            </li>
          </ul>
        </div>
      </section>
    </div>
  </AppShell>
</template>

<style scoped>
.responsive-table-wrap {
  overflow-x: auto;
  max-inline-size: 100%;
  container-type: inline-size;
}

@container (width < 560px) {
  .responsive-table,
  .responsive-table thead,
  .responsive-table tbody,
  .responsive-table tr,
  .responsive-table th,
  .responsive-table td {
    display: block;
  }

  .responsive-table thead {
    display: none;
  }

  .responsive-table tr {
    margin-block-end: 0.75rem;
    border: 1px solid var(--border-ledger);
    border-radius: 6px;
    padding: 0.5rem 0.75rem;
  }

  .responsive-table tbody th {
    padding: 0.25rem 0;
    border-bottom: 1px solid var(--border-ledger);
    margin-bottom: 0.35rem;
  }

  .responsive-table td {
    padding: 0.2rem 0;
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }

  .responsive-table td::before {
    content: attr(data-label) ": ";
    font-weight: 600;
    color: var(--ink-secondary);
  }
}
</style>
