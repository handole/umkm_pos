<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError, apiRequest, unwrapList } from '../../api/client'
import AppShell from '../../components/AppShell.vue'
import { useAuthStore } from '../../stores/auth'
import { formatDateTime, formatIDR } from '../../utils/format'

interface ShiftInfo {
  uuid: string
  shift_number: string
  status: string
  starting_cash: number
  expected_cash: number
  start_time: string
}

interface PaymentMethod {
  code: string
  name: string
  requires_reference: boolean
}

interface ProductVariant {
  id: number
  uuid: string
  sku: string
  variant_name: string
  base_sell_price: number
  uoms?: Array<{ uuid: string; uom_name: string; sell_price: number; is_base_uom: boolean }>
}

interface ProductItem {
  uuid: string
  name: string
  variants?: ProductVariant[]
}

interface CartLine {
  variant_uuid: string
  uom_uuid?: string
  name: string
  sku: string
  uom_name: string
  unit_price: number
  quantity: number
}

interface OrderRecord {
  uuid: string
  order_number: string
  total_amount: number
  status: string
  ordered_at: string
}

interface ReceiptData {
  order_number: string
  ordered_at: string
  cashier_name: string
  outlet: { name: string; address?: string }
  items: Array<{ name: string; qty: number; uom: string; subtotal: number }>
  summary: { subtotal: number; discount: number; tax: number; total: number; paid: number; change: number }
}

const auth = useAuthStore()
const canVoid = computed(() => auth.user?.role === 'SUPERADMIN' || auth.permissions.includes('pos:void'))
const canShift = computed(() => auth.user?.role === 'SUPERADMIN' || auth.permissions.includes('pos:shift'))

const activeTab = ref<'cashier' | 'orders' | 'shift'>('cashier')
const loading = ref(false)
const errorMsg = ref('')
const actionMsg = ref('')

const currentShift = ref<ShiftInfo | null>(null)
const paymentMethods = ref<PaymentMethod[]>([])
const products = ref<ProductItem[]>([])
const orders = ref<OrderRecord[]>([])

const searchQuery = ref('')
const cart = ref<CartLine[]>([])
const couponCode = ref('')
const discountAmount = ref(0)
const selectedPaymentCode = ref('CASH')
const paidAmount = ref(0)
const paymentRef = ref('')
const checkingOut = ref(false)

const shiftOpenCash = ref(200000)
const shiftCloseCash = ref(0)
const cashLogForm = reactive({ type: 'CASH_IN', amount: 0, reason: '' })

const receiptDialog = ref<HTMLDialogElement | null>(null)
const receiptData = ref<ReceiptData | null>(null)

const cartSubtotal = computed(() => cart.value.reduce((sum, it) => sum + it.unit_price * it.quantity, 0))
const cartGrandTotal = computed(() => Math.max(0, cartSubtotal.value - discountAmount.value))

const filteredProducts = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return products.value
  return products.value.filter(
    (p) =>
      p.name.toLowerCase().includes(q) ||
      p.variants?.some((v) => v.sku.toLowerCase().includes(q)),
  )
})

async function loadAll() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [pmRes, prodRes, ordRes] = await Promise.all([
      apiRequest<PaymentMethod[]>('/api/v1/pos/payment-methods', { method: 'GET' }, auth.accessToken),
      apiRequest<ProductItem[] | { items: ProductItem[] }>('/api/v1/products?limit=50', { method: 'GET' }, auth.accessToken),
      apiRequest<OrderRecord[] | { items: OrderRecord[] }>('/api/v1/pos/orders?limit=20', { method: 'GET' }, auth.accessToken),
    ])
    paymentMethods.value = pmRes || []
    if (paymentMethods.value.length && !paymentMethods.value.some((m) => m.code === selectedPaymentCode.value)) {
      selectedPaymentCode.value = paymentMethods.value[0]!.code
    }
    products.value = unwrapList(prodRes)
    orders.value = unwrapList(ordRes)

    if (canShift.value) {
      try {
        currentShift.value = await apiRequest<ShiftInfo>('/api/v1/pos/shifts/current', { method: 'GET' }, auth.accessToken)
      } catch {
        currentShift.value = null
      }
    }
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal memuat data POS.'
  } finally {
    loading.value = false
  }
}

function addToCart(p: ProductItem) {
  const v = p.variants?.[0]
  if (!v) return
  const baseUom = v.uoms?.find((u) => u.is_base_uom) || v.uoms?.[0]
  const unitPrice = baseUom ? baseUom.sell_price : v.base_sell_price
  const uomUuid = baseUom?.uuid
  const existing = cart.value.find((c) => c.variant_uuid === v.uuid && c.uom_uuid === uomUuid)
  if (existing) {
    existing.quantity += 1
  } else {
    cart.value.push({
      variant_uuid: v.uuid,
      uom_uuid: uomUuid,
      name: `${p.name} (${v.variant_name})`,
      sku: v.sku,
      uom_name: baseUom?.uom_name || 'Pcs',
      unit_price: unitPrice,
      quantity: 1,
    })
  }
  paidAmount.value = cartGrandTotal.value
}

function updateQty(idx: number, delta: number) {
  const line = cart.value[idx]
  if (!line) return
  line.quantity += delta
  if (line.quantity <= 0) {
    cart.value.splice(idx, 1)
  }
  paidAmount.value = cartGrandTotal.value
}

async function handleEvaluatePromo() {
  if (!cart.value.length) return
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    const res = await apiRequest<{ total_discount: number; applied_promos?: Array<{ name: string }> }>(
      '/api/v1/promotions/evaluate',
      {
        method: 'POST',
        body: JSON.stringify({
          outlet_id: auth.outlets[0]?.id || 1,
          coupon_code: couponCode.value.trim() || undefined,
          items: cart.value.map((c) => ({
            variant_uuid: c.variant_uuid,
            quantity: c.quantity,
            unit_price: c.unit_price,
          })),
        }),
      },
      auth.accessToken,
    )
    discountAmount.value = res.total_discount || 0
    paidAmount.value = cartGrandTotal.value
    actionMsg.value =
      discountAmount.value > 0
        ? `Diskon promo diterapkan: ${formatIDR(discountAmount.value)}`
        : 'Tidak ada promo yang memenuhi syarat keranjang saat ini.'
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal mengevaluasi promo.'
  }
}

async function handleCheckout() {
  if (!cart.value.length) return
  checkingOut.value = true
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    const res = await apiRequest<OrderRecord>(
      '/api/v1/pos/checkout',
      {
        method: 'POST',
        body: JSON.stringify({
          idempotency_key: crypto.randomUUID(),
          outlet_id: auth.outlets[0]?.id || 1,
          salesperson_user_id: auth.user?.id,
          discount_amount: discountAmount.value,
          coupon_code: couponCode.value.trim() || undefined,
          items: cart.value.map((c) => ({
            variant_uuid: c.variant_uuid,
            uom_uuid: c.uom_uuid,
            quantity: c.quantity,
          })),
          payments: [
            {
              payment_method_code: selectedPaymentCode.value,
              amount: Number(paidAmount.value || cartGrandTotal.value),
              reference_number: paymentRef.value.trim() || undefined,
            },
          ],
        }),
      },
      auth.accessToken,
    )
    actionMsg.value = `Transaksi ${res.order_number} berhasil (${formatIDR(res.total_amount)}).`
    cart.value = []
    couponCode.value = ''
    discountAmount.value = 0
    paidAmount.value = 0
    paymentRef.value = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Checkout gagal diproses.'
  } finally {
    checkingOut.value = false
  }
}

async function openReceipt(order: OrderRecord) {
  errorMsg.value = ''
  try {
    receiptData.value = await apiRequest<ReceiptData>(
      `/api/v1/pos/orders/${order.uuid}/receipt`,
      { method: 'GET' },
      auth.accessToken,
    )
    receiptDialog.value?.showModal()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal memuat struk.'
  }
}

async function handleVoidOrder(order: OrderRecord) {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      `/api/v1/pos/orders/${order.uuid}/void`,
      { method: 'POST', body: JSON.stringify({ reason: 'Pembatalan oleh supervisor dari Web Admin' }) },
      auth.accessToken,
    )
    actionMsg.value = `Order ${order.order_number} dibatalkan dan stok dikembalikan.`
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal membatalkan order.'
  }
}

async function handleOpenShift() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/pos/shifts/open',
      { method: 'POST', body: JSON.stringify({ outlet_id: auth.outlets[0]?.id || 1, starting_cash: Number(shiftOpenCash.value) }) },
      auth.accessToken,
    )
    actionMsg.value = 'Shift kasir berhasil dibuka.'
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal membuka shift.'
  }
}

async function handleCashLog() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest('/api/v1/pos/shifts/cash-log', { method: 'POST', body: JSON.stringify(cashLogForm) }, auth.accessToken)
    actionMsg.value = 'Catatan kas laci berhasil disimpan.'
    cashLogForm.amount = 0
    cashLogForm.reason = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal mencatat kas laci.'
  }
}

async function handleCloseShift() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/pos/shifts/close',
      { method: 'POST', body: JSON.stringify({ actual_cash: Number(shiftCloseCash.value) }) },
      auth.accessToken,
    )
    actionMsg.value = 'Shift kasir berhasil ditutup.'
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menutup shift.'
  }
}

onMounted(loadAll)
</script>

<template>
  <AppShell page-title="Kasir POS &amp; Shift" page-subtitle="Checkout transaksi, manajemen laci kasir, cetak struk, dan otorisasi void.">
    <div v-if="errorMsg" role="alert" class="mb-4 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-4 py-3 text-sm font-medium text-[var(--error-text)]">
      Error: {{ errorMsg }}
    </div>
    <div v-if="actionMsg" role="status" class="mb-4 rounded-md border border-[var(--brand-forest)] bg-[var(--bg-surface)] px-4 py-3 text-sm font-semibold text-[var(--brand-forest)]">
      {{ actionMsg }}
    </div>

    <div role="tablist" aria-label="Menu POS" class="mb-5 flex flex-wrap gap-2 rounded-lg border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-1 w-fit">
      <button
        type="button"
        role="tab"
        :aria-selected="activeTab === 'cashier'"
        class="min-h-[44px] rounded-md px-4 py-2 text-sm font-semibold"
        :class="activeTab === 'cashier' ? 'bg-[var(--brand-forest)] text-white' : 'text-[var(--ink-primary)]'"
        @click="activeTab = 'cashier'"
      >
        Layar Kasir
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="activeTab === 'orders'"
        class="min-h-[44px] rounded-md px-4 py-2 text-sm font-semibold"
        :class="activeTab === 'orders' ? 'bg-[var(--brand-forest)] text-white' : 'text-[var(--ink-primary)]'"
        @click="activeTab = 'orders'"
      >
        Riwayat Transaksi ({{ orders.length }})
      </button>
      <button
        v-if="canShift"
        type="button"
        role="tab"
        :aria-selected="activeTab === 'shift'"
        class="min-h-[44px] rounded-md px-4 py-2 text-sm font-semibold"
        :class="activeTab === 'shift' ? 'bg-[var(--brand-forest)] text-white' : 'text-[var(--ink-primary)]'"
        @click="activeTab = 'shift'"
      >
        Shift Laci
      </button>
    </div>

    <div v-if="loading" role="status" class="py-6 text-sm text-[var(--ink-secondary)]">Memuat terminal POS...</div>

    <!-- Cashier Split View -->
    <div v-else-if="activeTab === 'cashier'" class="grid grid-cols-1 gap-6 lg:grid-cols-12">
      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5 lg:col-span-7">
        <div class="mb-4">
          <label for="pos-search" class="visually-hidden">Cari produk</label>
          <input
            id="pos-search"
            v-model="searchQuery"
            type="search"
            placeholder="Cari nama produk atau SKU..."
            class="field-input"
          />
        </div>

        <p v-if="!filteredProducts.length" class="text-sm text-[var(--ink-secondary)]">
          Tidak ada produk yang cocok atau katalog masih kosong.
        </p>
        <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <button
            v-for="p in filteredProducts"
            :key="p.uuid"
            type="button"
            class="flex min-h-[72px] flex-col justify-between rounded-md border border-[var(--border-ledger)] bg-[var(--bg-paper)] p-3.5 text-left hover:border-[var(--brand-forest)]"
            @click="addToCart(p)"
          >
            <span class="text-sm font-bold text-[var(--ink-primary)]">{{ p.name }}</span>
            <span class="mt-1 flex items-center justify-between text-xs text-[var(--ink-secondary)]">
              <span>{{ p.variants?.[0]?.sku || '-' }}</span>
              <span class="font-bold text-[var(--brand-forest)]">
                {{ formatIDR(p.variants?.[0]?.base_sell_price) }}
              </span>
            </span>
          </button>
        </div>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5 lg:col-span-5">
        <h2 class="text-base font-bold">Keranjang Pembayaran</h2>
        <p v-if="!cart.length" class="mt-4 text-sm text-[var(--ink-secondary)]">
          Klik produk di sebelah kiri untuk menambahkan ke keranjang.
        </p>

        <div v-else class="mt-4 space-y-4">
          <ul class="divide-y divide-[var(--border-ledger)] text-sm">
            <li v-for="(line, idx) in cart" :key="line.variant_uuid" class="flex items-center justify-between gap-2 py-2.5">
              <div>
                <p class="font-semibold">{{ line.name }}</p>
                <p class="text-xs text-[var(--ink-secondary)]">{{ formatIDR(line.unit_price) }} / {{ line.uom_name }}</p>
              </div>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  aria-label="Kurangi jumlah"
                  class="min-h-[36px] min-w-[36px] rounded border border-[var(--border-strong)] font-bold"
                  @click="updateQty(idx, -1)"
                >
                  -
                </button>
                <span class="w-7 text-center font-semibold">{{ line.quantity }}</span>
                <button
                  type="button"
                  aria-label="Tambah jumlah"
                  class="min-h-[36px] min-w-[36px] rounded border border-[var(--border-strong)] font-bold"
                  @click="updateQty(idx, 1)"
                >
                  +
                </button>
              </div>
            </li>
          </ul>

          <div class="flex gap-2">
            <label for="pos-coupon" class="visually-hidden">Kode Kupon</label>
            <input id="pos-coupon" v-model="couponCode" placeholder="Kode kupon (opsional)" class="field-input flex-1" />
            <button type="button" class="min-h-[48px] rounded-md border border-[var(--border-strong)] bg-[var(--bg-subtle)] px-3 text-xs font-semibold" @click="handleEvaluatePromo">
              Cek Promo
            </button>
          </div>

          <div class="space-y-1.5 border-t border-[var(--border-ledger)] pt-3 text-sm">
            <div class="flex justify-between">
              <span class="text-[var(--ink-secondary)]">Subtotal</span>
              <span>{{ formatIDR(cartSubtotal) }}</span>
            </div>
            <div v-if="discountAmount > 0" class="flex justify-between text-[var(--accent-terracotta)]">
              <span>Diskon Promo</span>
              <span>-{{ formatIDR(discountAmount) }}</span>
            </div>
            <div class="flex justify-between text-base font-bold text-[var(--brand-forest)]">
              <span>Total Tagihan</span>
              <span>{{ formatIDR(cartGrandTotal) }}</span>
            </div>
          </div>

          <form class="space-y-3 border-t border-[var(--border-ledger)] pt-3" @submit.prevent="handleCheckout">
            <div>
              <label for="pos-pay-method" class="mb-1 block text-xs font-semibold">Metode Pembayaran</label>
              <select id="pos-pay-method" v-model="selectedPaymentCode" class="field-input">
                <option v-for="pm in paymentMethods" :key="pm.code" :value="pm.code">{{ pm.name }}</option>
              </select>
            </div>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label for="pos-paid" class="mb-1 block text-xs font-semibold">Nominal Bayar</label>
                <input id="pos-paid" v-model.number="paidAmount" type="number" min="0" required class="field-input" />
              </div>
              <div>
                <label for="pos-ref" class="mb-1 block text-xs font-semibold">No. Referensi (Opsional)</label>
                <input id="pos-ref" v-model="paymentRef" class="field-input" />
              </div>
            </div>
            <button
              type="submit"
              :disabled="checkingOut"
              class="min-h-[48px] w-full rounded-md bg-[var(--brand-forest)] px-4 py-3 text-base font-semibold text-white hover:bg-[var(--brand-forest-hover)] disabled:opacity-65"
            >
              {{ checkingOut ? 'Memproses Pembayaran...' : `Bayar ${formatIDR(cartGrandTotal)}` }}
            </button>
          </form>
        </div>
      </section>
    </div>

    <!-- Orders Tab -->
    <section v-else-if="activeTab === 'orders'" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
      <h2 class="mb-3 text-base font-bold">Riwayat Transaksi</h2>
      <p v-if="!orders.length" class="text-sm text-[var(--ink-secondary)]">Belum ada transaksi tercatat.</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
              <th scope="col" class="px-3 py-2.5">Nomor Order</th>
              <th scope="col" class="px-3 py-2.5">Waktu</th>
              <th scope="col" class="px-3 py-2.5">Total</th>
              <th scope="col" class="px-3 py-2.5">Status</th>
              <th scope="col" class="px-3 py-2.5">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in orders" :key="o.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
              <th scope="row" class="px-3 py-2.5 font-semibold">{{ o.order_number }}</th>
              <td class="px-3 py-2.5 text-[var(--ink-secondary)]">{{ formatDateTime(o.ordered_at) }}</td>
              <td class="px-3 py-2.5 font-semibold">{{ formatIDR(o.total_amount) }}</td>
              <td class="px-3 py-2.5">{{ o.status }}</td>
              <td class="px-3 py-2.5 flex flex-wrap gap-2">
                <button type="button" class="min-h-[36px] rounded border border-[var(--border-strong)] px-2.5 py-1 text-xs font-semibold" @click="openReceipt(o)">
                  Struk
                </button>
                <button
                  v-if="canVoid && o.status === 'COMPLETED'"
                  type="button"
                  class="min-h-[36px] rounded border border-[var(--error-text)] px-2.5 py-1 text-xs font-semibold text-[var(--error-text)]"
                  @click="handleVoidOrder(o)"
                >
                  Void
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Shift Tab -->
    <div v-else class="grid grid-cols-1 gap-6 md:grid-cols-2">
      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Status Shift Aktif</h2>
        <div v-if="currentShift" class="mt-4 space-y-3 text-sm">
          <p>Nomor Shift: <strong>{{ currentShift.shift_number }}</strong></p>
          <p>Modal Awal: <strong>{{ formatIDR(currentShift.starting_cash) }}</strong></p>
          <p>Perkiraan Kas Laci: <strong class="text-[var(--brand-forest)]">{{ formatIDR(currentShift.expected_cash) }}</strong></p>

          <form class="space-y-3 border-t border-[var(--border-ledger)] pt-4" @submit.prevent="handleCloseShift">
            <label for="close-cash" class="block text-xs font-semibold">Uang Fisik Aktual Saat Tutup Shift</label>
            <input id="close-cash" v-model.number="shiftCloseCash" type="number" min="0" required class="field-input" />
            <button type="submit" class="min-h-[44px] rounded-md bg-[var(--accent-terracotta)] px-4 py-2 text-sm font-semibold text-white">
              Tutup Shift
            </button>
          </form>
        </div>

        <form v-else class="mt-4 space-y-3" @submit.prevent="handleOpenShift">
          <p class="text-sm text-[var(--ink-secondary)]">Belum ada shift terbuka.</p>
          <label for="open-cash" class="block text-xs font-semibold">Modal Awal Laci (Rp)</label>
          <input id="open-cash" v-model.number="shiftOpenCash" type="number" min="0" required class="field-input" />
          <button type="submit" class="min-h-[44px] rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
            Buka Shift Kasir
          </button>
        </form>
      </section>

      <section v-if="currentShift" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Catat Kas Masuk / Keluar (Petty Cash)</h2>
        <form class="mt-4 space-y-3" @submit.prevent="handleCashLog">
          <div>
            <label for="cl-type" class="mb-1 block text-xs font-semibold">Tipe</label>
            <select id="cl-type" v-model="cashLogForm.type" class="field-input">
              <option value="CASH_IN">CASH_IN (Tambah Laci)</option>
              <option value="CASH_OUT">CASH_OUT (Ambil Laci)</option>
            </select>
          </div>
          <div>
            <label for="cl-amt" class="mb-1 block text-xs font-semibold">Nominal (Rp)</label>
            <input id="cl-amt" v-model.number="cashLogForm.amount" type="number" min="1" required class="field-input" />
          </div>
          <div>
            <label for="cl-reason" class="mb-1 block text-xs font-semibold">Keterangan</label>
            <input id="cl-reason" v-model="cashLogForm.reason" required class="field-input" />
          </div>
          <button type="submit" class="min-h-[44px] rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
            Simpan Log Kas
          </button>
        </form>
      </section>
    </div>

    <!-- Native <dialog> Receipt Preview -->
    <dialog
      ref="receiptDialog"
      aria-label="Preview Struk Belanja"
      class="m-auto w-[min(420px,92vw)] rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5 backdrop:bg-black/45"
    >
      <div v-if="receiptData" class="space-y-3 text-sm">
        <div class="flex items-center justify-between border-b border-[var(--border-ledger)] pb-2">
          <h3 class="font-bold">{{ receiptData.outlet?.name || 'UMKM POS' }}</h3>
          <button type="button" class="min-h-[36px] rounded border border-[var(--border-strong)] px-2.5 py-1 text-xs font-semibold" @click="receiptDialog?.close()">
            Tutup
          </button>
        </div>
        <p class="text-xs text-[var(--ink-secondary)]">{{ receiptData.order_number }} | {{ formatDateTime(receiptData.ordered_at) }}</p>
        <ul class="divide-y divide-[var(--border-ledger)]">
          <li v-for="(it, i) in receiptData.items" :key="i" class="flex justify-between py-1.5">
            <span>{{ it.name }} ({{ it.qty }} {{ it.uom }})</span>
            <span class="font-semibold">{{ formatIDR(it.subtotal) }}</span>
          </li>
        </ul>
        <div class="space-y-1 border-t border-[var(--border-ledger)] pt-2 font-semibold">
          <div class="flex justify-between"><span>Total</span><span>{{ formatIDR(receiptData.summary.total) }}</span></div>
          <div class="flex justify-between text-xs text-[var(--ink-secondary)]"><span>Dibayar</span><span>{{ formatIDR(receiptData.summary.paid) }}</span></div>
          <div class="flex justify-between text-xs text-[var(--ink-secondary)]"><span>Kembali</span><span>{{ formatIDR(receiptData.summary.change) }}</span></div>
        </div>
      </div>
    </dialog>
  </AppShell>
</template>
