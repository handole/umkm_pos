<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError, apiRequest, unwrapList } from '../../api/client'
import AppShell from '../../components/AppShell.vue'
import { useAuthStore } from '../../stores/auth'
import { formatDateTime, formatIDR } from '../../utils/format'

interface LocationItem {
  id: number
  uuid: string
  outlet_id: number
  code: string
  name: string
  type: string
}

interface StockItem {
  uuid: string
  location_name: string
  variant_sku: string
  variant_name: string
  product_name: string
  quantity: number
}

interface MovementItem {
  uuid: string
  location_name: string
  variant_sku: string
  movement_type: string
  quantity_change: number
  balance_after: number
  reference_number: string
  notes: string
  created_at: string
}

interface SupplierItem {
  id: number
  uuid: string
  code: string
  name: string
  phone?: string
}

interface PurchaseOrderItem {
  uuid: string
  po_number: string
  status: string
  total_amount: number
  order_date: string
  supplier?: { name: string }
  destination_location?: { name: string }
  items?: Array<{
    uuid: string
    quantity_ordered: number
    quantity_received: number
    unit_cost: number
    variant?: { sku: string; variant_name: string }
  }>
}

interface ProductVariantFlat {
  uuid: string
  sku: string
  label: string
}

const auth = useAuthStore()
const canTransfer = computed(() => auth.user?.role === 'SUPERADMIN' || auth.permissions.includes('inventory:transfer'))
const canApprove = computed(() => auth.user?.role === 'SUPERADMIN' || auth.permissions.includes('inventory:approve-opname'))

const activeTab = ref<'stocks' | 'movements' | 'locations' | 'purchasing'>('stocks')
const loading = ref(false)
const errorMsg = ref('')
const actionMsg = ref('')

const locations = ref<LocationItem[]>([])
const stocks = ref<StockItem[]>([])
const movements = ref<MovementItem[]>([])
const suppliers = ref<SupplierItem[]>([])
const purchaseOrders = ref<PurchaseOrderItem[]>([])
const variants = ref<ProductVariantFlat[]>([])

const locForm = reactive({ outlet_id: 1, code: '', name: '', type: 'WAREHOUSE' })
const adjForm = reactive({ location_uuid: '', variant_uuid: '', quantity_change: 0, cost_price: 0, notes: '' })
const supForm = reactive({ code: '', name: '', phone: '' })
const poForm = reactive({
  outlet_id: 1,
  supplier_uuid: '',
  destination_location_uuid: '',
  order_date: new Date().toISOString().slice(0, 10),
  variant_uuid: '',
  quantity_ordered: 10,
  unit_cost: 0,
})

async function loadAll() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [locRes, stkRes, movRes, supRes, poRes, prodRes] = await Promise.all([
      apiRequest<LocationItem[]>('/api/v1/inventory/locations', { method: 'GET' }, auth.accessToken),
      apiRequest<StockItem[] | { items: StockItem[] }>('/api/v1/inventory/stocks?limit=50', { method: 'GET' }, auth.accessToken),
      apiRequest<MovementItem[] | { items: MovementItem[] }>('/api/v1/inventory/movements?limit=30', { method: 'GET' }, auth.accessToken),
      apiRequest<SupplierItem[]>('/api/v1/inventory/suppliers', { method: 'GET' }, auth.accessToken),
      apiRequest<PurchaseOrderItem[] | { items: PurchaseOrderItem[] }>('/api/v1/inventory/purchase-orders?limit=30', { method: 'GET' }, auth.accessToken),
      apiRequest<
        | Array<{ name: string; variants?: Array<{ uuid: string; sku: string; variant_name: string }> }>
        | { items: Array<{ name: string; variants?: Array<{ uuid: string; sku: string; variant_name: string }> }> }
      >(
        '/api/v1/products?limit=50',
        { method: 'GET' },
        auth.accessToken,
      ),
    ])
    locations.value = locRes || []
    stocks.value = unwrapList(stkRes)
    movements.value = unwrapList(movRes)
    suppliers.value = supRes || []
    purchaseOrders.value = unwrapList(poRes)

    const flat: ProductVariantFlat[] = []
    for (const p of unwrapList(prodRes)) {
      for (const v of p.variants || []) {
        flat.push({ uuid: v.uuid, sku: v.sku, label: `${p.name} - ${v.variant_name} (${v.sku})` })
      }
    }
    variants.value = flat
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal memuat data inventaris.'
  } finally {
    loading.value = false
  }
}

async function handleCreateLocation() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest('/api/v1/inventory/locations', { method: 'POST', body: JSON.stringify(locForm) }, auth.accessToken)
    actionMsg.value = `Lokasi "${locForm.name}" berhasil ditambahkan.`
    locForm.code = ''
    locForm.name = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menambah lokasi.'
  }
}

async function handleAdjustment() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest('/api/v1/inventory/adjustments', { method: 'POST', body: JSON.stringify(adjForm) }, auth.accessToken)
    actionMsg.value = 'Penyesuaian stok berhasil dicatat ke kartu stok.'
    adjForm.quantity_change = 0
    adjForm.notes = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menyesuaikan stok.'
  }
}

async function handleCreateSupplier() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest('/api/v1/inventory/suppliers', { method: 'POST', body: JSON.stringify(supForm) }, auth.accessToken)
    actionMsg.value = `Supplier "${supForm.name}" berhasil didaftarkan.`
    supForm.code = ''
    supForm.name = ''
    supForm.phone = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menambah supplier.'
  }
}

async function handleCreatePO() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/inventory/purchase-orders',
      {
        method: 'POST',
        body: JSON.stringify({
          outlet_id: poForm.outlet_id,
          supplier_uuid: poForm.supplier_uuid,
          destination_location_uuid: poForm.destination_location_uuid,
          order_date: poForm.order_date,
          items: [
            {
              variant_uuid: poForm.variant_uuid,
              quantity_ordered: Number(poForm.quantity_ordered),
              unit_cost: Number(poForm.unit_cost),
            },
          ],
        }),
      },
      auth.accessToken,
    )
    actionMsg.value = 'Purchase Order berhasil dibuat.'
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal membuat Purchase Order.'
  }
}

async function handleReceiveAllPO(po: PurchaseOrderItem) {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    const detail = await apiRequest<PurchaseOrderItem>(
      `/api/v1/inventory/purchase-orders/${po.uuid}`,
      { method: 'GET' },
      auth.accessToken,
    )
    const itemsToReceive = (detail.items || [])
      .filter((it) => it.quantity_ordered > it.quantity_received)
      .map((it) => ({
        item_uuid: it.uuid,
        quantity_received: it.quantity_ordered - it.quantity_received,
      }))
    if (!itemsToReceive.length) return
    await apiRequest(
      `/api/v1/inventory/purchase-orders/${po.uuid}/receive`,
      { method: 'POST', body: JSON.stringify({ items: itemsToReceive }) },
      auth.accessToken,
    )
    actionMsg.value = `PO ${po.po_number} diterima penuh dan stok diperbarui.`
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menerima PO.'
  }
}

onMounted(loadAll)
</script>

<template>
  <AppShell page-title="Stok, Gudang &amp; Pembelian" page-subtitle="Saldo real-time, kartu stok immutable, lokasi penyimpanan, dan Purchase Order.">
    <template #header-actions>
      <button type="button" class="min-h-[44px] rounded-md border border-[var(--border-strong)] bg-[var(--bg-surface)] px-4 py-2 text-sm font-semibold hover:bg-[var(--bg-subtle)]" @click="loadAll">
        Muat Ulang
      </button>
    </template>

    <div v-if="errorMsg" role="alert" class="mb-4 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-4 py-3 text-sm font-medium text-[var(--error-text)]">
      Error: {{ errorMsg }}
    </div>
    <div v-if="actionMsg" role="status" class="mb-4 rounded-md border border-[var(--brand-forest)] bg-[var(--bg-surface)] px-4 py-3 text-sm font-semibold text-[var(--brand-forest)]">
      {{ actionMsg }}
    </div>

    <div role="tablist" aria-label="Menu inventaris" class="mb-5 flex flex-wrap gap-2 rounded-lg border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-1">
      <button
        v-for="tab in [
          { id: 'stocks', label: `Saldo Stok (${stocks.length})` },
          { id: 'movements', label: 'Kartu Stok' },
          { id: 'locations', label: `Lokasi (${locations.length})` },
          { id: 'purchasing', label: `Supplier & PO (${purchaseOrders.length})` },
        ]"
        :key="tab.id"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.id"
        class="min-h-[44px] rounded-md px-3.5 py-2 text-sm font-semibold"
        :class="activeTab === tab.id ? 'bg-[var(--brand-forest)] text-white' : 'text-[var(--ink-primary)]'"
        @click="activeTab = tab.id as any"
      >
        {{ tab.label }}
      </button>
    </div>

    <div v-if="loading" role="status" class="py-6 text-sm text-[var(--ink-secondary)]">Memuat data inventaris...</div>

    <!-- Stocks + Manual Adjustment -->
    <div v-else-if="activeTab === 'stocks'" class="space-y-6">
      <section v-if="canApprove" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Penyesuaian Stok Manual</h2>
        <form class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-5" @submit.prevent="handleAdjustment">
          <div>
            <label for="adj-loc" class="mb-1 block text-xs font-semibold">Lokasi</label>
            <select id="adj-loc" v-model="adjForm.location_uuid" required class="field-input">
              <option value="" disabled>Pilih Lokasi</option>
              <option v-for="l in locations" :key="l.uuid" :value="l.uuid">{{ l.name }}</option>
            </select>
          </div>
          <div>
            <label for="adj-var" class="mb-1 block text-xs font-semibold">Varian</label>
            <select id="adj-var" v-model="adjForm.variant_uuid" required class="field-input">
              <option value="" disabled>Pilih Varian</option>
              <option v-for="v in variants" :key="v.uuid" :value="v.uuid">{{ v.label }}</option>
            </select>
          </div>
          <div>
            <label for="adj-qty" class="mb-1 block text-xs font-semibold">Perubahan (+/-)</label>
            <input id="adj-qty" v-model.number="adjForm.quantity_change" type="number" required class="field-input" />
          </div>
          <div>
            <label for="adj-notes" class="mb-1 block text-xs font-semibold">Catatan</label>
            <input id="adj-notes" v-model="adjForm.notes" required placeholder="Misal: Saldo awal" class="field-input" />
          </div>
          <div class="flex items-end">
            <button type="submit" class="min-h-[48px] w-full rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
              Simpan
            </button>
          </div>
        </form>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Saldo Stok Real-Time</h2>
        <p v-if="!stocks.length" class="text-sm text-[var(--ink-secondary)]">Belum ada saldo stok tercatat.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
                <th scope="col" class="px-3 py-2.5">SKU</th>
                <th scope="col" class="px-3 py-2.5">Produk &amp; Varian</th>
                <th scope="col" class="px-3 py-2.5">Lokasi</th>
                <th scope="col" class="px-3 py-2.5">Saldo</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in stocks" :key="s.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
                <th scope="row" class="px-3 py-2.5 font-semibold">{{ s.variant_sku }}</th>
                <td class="px-3 py-2.5">{{ s.product_name }} - {{ s.variant_name }}</td>
                <td class="px-3 py-2.5 text-[var(--ink-secondary)]">{{ s.location_name }}</td>
                <td class="px-3 py-2.5 font-bold text-[var(--brand-forest)]">{{ s.quantity }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <!-- Movements Ledger -->
    <section v-else-if="activeTab === 'movements'" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
      <h2 class="mb-3 text-base font-bold">Kartu Stok (Mutasi Ledger)</h2>
      <p v-if="!movements.length" class="text-sm text-[var(--ink-secondary)]">Belum ada riwayat mutasi stok.</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
              <th scope="col" class="px-3 py-2.5">Waktu</th>
              <th scope="col" class="px-3 py-2.5">SKU</th>
              <th scope="col" class="px-3 py-2.5">Lokasi</th>
              <th scope="col" class="px-3 py-2.5">Tipe</th>
              <th scope="col" class="px-3 py-2.5">Perubahan</th>
              <th scope="col" class="px-3 py-2.5">Saldo Akhir</th>
              <th scope="col" class="px-3 py-2.5">Referensi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in movements" :key="m.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
              <td class="px-3 py-2.5 text-[var(--ink-secondary)]">{{ formatDateTime(m.created_at) }}</td>
              <th scope="row" class="px-3 py-2.5 font-semibold">{{ m.variant_sku }}</th>
              <td class="px-3 py-2.5">{{ m.location_name }}</td>
              <td class="px-3 py-2.5 font-semibold">{{ m.movement_type }}</td>
              <td class="px-3 py-2.5 font-bold" :class="m.quantity_change >= 0 ? 'text-[var(--brand-forest)]' : 'text-[var(--error-text)]'">
                {{ m.quantity_change > 0 ? `+${m.quantity_change}` : m.quantity_change }}
              </td>
              <td class="px-3 py-2.5">{{ m.balance_after }}</td>
              <td class="px-3 py-2.5 text-xs text-[var(--ink-secondary)]">{{ m.reference_number || m.notes }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Locations -->
    <div v-else-if="activeTab === 'locations'" class="space-y-6">
      <section v-if="canTransfer" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Tambah Lokasi Penyimpanan</h2>
        <form class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-4" @submit.prevent="handleCreateLocation">
          <div>
            <label for="loc-code" class="mb-1 block text-xs font-semibold">Kode</label>
            <input id="loc-code" v-model="locForm.code" required placeholder="WH-01" class="field-input" />
          </div>
          <div>
            <label for="loc-name" class="mb-1 block text-xs font-semibold">Nama Lokasi</label>
            <input id="loc-name" v-model="locForm.name" required placeholder="Gudang Utama" class="field-input" />
          </div>
          <div>
            <label for="loc-type" class="mb-1 block text-xs font-semibold">Tipe</label>
            <select id="loc-type" v-model="locForm.type" class="field-input">
              <option value="WAREHOUSE">WAREHOUSE</option>
              <option value="STORE_FRONT">STORE_FRONT</option>
              <option value="TRANSIT">TRANSIT</option>
              <option value="DAMAGED">DAMAGED</option>
            </select>
          </div>
          <div class="flex items-end">
            <button type="submit" class="min-h-[48px] w-full rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
              Tambah Lokasi
            </button>
          </div>
        </form>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Daftar Lokasi</h2>
        <p v-if="!locations.length" class="text-sm text-[var(--ink-secondary)]">Belum ada lokasi.</p>
        <ul v-else class="divide-y divide-[var(--border-ledger)] text-sm">
          <li v-for="l in locations" :key="l.uuid" class="flex items-center justify-between py-2.5">
            <div>
              <span class="font-semibold">{{ l.name }}</span>
              <span class="ml-2 text-xs text-[var(--ink-secondary)]">({{ l.code }})</span>
            </div>
            <span class="rounded border border-[var(--border-ledger)] bg-[var(--bg-paper)] px-2 py-0.5 text-xs font-semibold">{{ l.type }}</span>
          </li>
        </ul>
      </section>
    </div>

    <!-- Suppliers & Purchase Orders -->
    <div v-else class="space-y-6">
      <div v-if="canTransfer" class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
          <h2 class="text-base font-bold">Tambah Supplier</h2>
          <form class="mt-3 space-y-3" @submit.prevent="handleCreateSupplier">
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label for="sup-code" class="mb-1 block text-xs font-semibold">Kode Supplier</label>
                <input id="sup-code" v-model="supForm.code" required class="field-input" />
              </div>
              <div>
                <label for="sup-name" class="mb-1 block text-xs font-semibold">Nama Supplier</label>
                <input id="sup-name" v-model="supForm.name" required class="field-input" />
              </div>
            </div>
            <div>
              <label for="sup-phone" class="mb-1 block text-xs font-semibold">Telepon</label>
              <input id="sup-phone" v-model="supForm.phone" type="tel" class="field-input" />
            </div>
            <button type="submit" class="min-h-[44px] rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
              Simpan Supplier
            </button>
          </form>
        </section>

        <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
          <h2 class="text-base font-bold">Buat Purchase Order (PO)</h2>
          <form class="mt-3 space-y-3" @submit.prevent="handleCreatePO">
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label for="po-sup" class="mb-1 block text-xs font-semibold">Supplier</label>
                <select id="po-sup" v-model="poForm.supplier_uuid" required class="field-input">
                  <option value="" disabled>Pilih Supplier</option>
                  <option v-for="s in suppliers" :key="s.uuid" :value="s.uuid">{{ s.name }}</option>
                </select>
              </div>
              <div>
                <label for="po-loc" class="mb-1 block text-xs font-semibold">Lokasi Tujuan</label>
                <select id="po-loc" v-model="poForm.destination_location_uuid" required class="field-input">
                  <option value="" disabled>Pilih Lokasi</option>
                  <option v-for="l in locations" :key="l.uuid" :value="l.uuid">{{ l.name }}</option>
                </select>
              </div>
            </div>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div>
                <label for="po-var" class="mb-1 block text-xs font-semibold">Varian</label>
                <select id="po-var" v-model="poForm.variant_uuid" required class="field-input">
                  <option value="" disabled>Pilih Varian</option>
                  <option v-for="v in variants" :key="v.uuid" :value="v.uuid">{{ v.label }}</option>
                </select>
              </div>
              <div>
                <label for="po-qty" class="mb-1 block text-xs font-semibold">Qty Pesan</label>
                <input id="po-qty" v-model.number="poForm.quantity_ordered" type="number" min="1" required class="field-input" />
              </div>
              <div>
                <label for="po-cost" class="mb-1 block text-xs font-semibold">Harga Beli/Unit</label>
                <input id="po-cost" v-model.number="poForm.unit_cost" type="number" min="0" required class="field-input" />
              </div>
            </div>
            <button type="submit" class="min-h-[44px] rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
              Terbitkan PO
            </button>
          </form>
        </section>
      </div>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Daftar Purchase Order</h2>
        <p v-if="!purchaseOrders.length" class="text-sm text-[var(--ink-secondary)]">Belum ada Purchase Order.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
                <th scope="col" class="px-3 py-2.5">Nomor PO</th>
                <th scope="col" class="px-3 py-2.5">Supplier</th>
                <th scope="col" class="px-3 py-2.5">Lokasi</th>
                <th scope="col" class="px-3 py-2.5">Total</th>
                <th scope="col" class="px-3 py-2.5">Status</th>
                <th scope="col" class="px-3 py-2.5">Aksi</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="po in purchaseOrders" :key="po.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
                <th scope="row" class="px-3 py-2.5 font-semibold">{{ po.po_number }}</th>
                <td class="px-3 py-2.5">{{ po.supplier?.name || '-' }}</td>
                <td class="px-3 py-2.5">{{ po.destination_location?.name || '-' }}</td>
                <td class="px-3 py-2.5 font-semibold">{{ formatIDR(po.total_amount) }}</td>
                <td class="px-3 py-2.5">{{ po.status }}</td>
                <td class="px-3 py-2.5">
                  <button
                    v-if="canTransfer && po.status !== 'RECEIVED' && po.status !== 'CANCELLED'"
                    type="button"
                    class="min-h-[36px] rounded bg-[var(--brand-forest)] px-3 py-1 text-xs font-semibold text-white"
                    @click="handleReceiveAllPO(po)"
                  >
                    Terima Barang
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppShell>
</template>
