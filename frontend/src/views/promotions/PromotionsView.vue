<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError, apiRequest, unwrapList } from '../../api/client'
import AppShell from '../../components/AppShell.vue'
import { useAuthStore } from '../../stores/auth'
import { formatDate, formatIDR } from '../../utils/format'

interface CustomerItem {
  id: number
  uuid: string
  code: string
  name: string
  phone: string
  email?: string
  member_tier: string
  total_spend: number
}

interface PromotionItem {
  id: number
  uuid: string
  code: string
  name: string
  promo_type: string
  discount_type: string
  discount_value: number
  is_active: boolean
  start_date: string
  end_date: string
}

const auth = useAuthStore()
const canManagePromo = computed(() => auth.user?.role === 'SUPERADMIN' || auth.permissions.includes('promotion:manage'))

const activeTab = ref<'promotions' | 'customers'>('promotions')
const loading = ref(false)
const errorMsg = ref('')
const actionMsg = ref('')

const promotions = ref<PromotionItem[]>([])
const customers = ref<CustomerItem[]>([])

const custForm = reactive({ code: '', name: '', phone: '', email: '', member_tier: 'REGULAR' })
const promoForm = reactive({
  code: '',
  name: '',
  promo_type: 'COUPON_CODE',
  discount_type: 'PERCENTAGE',
  discount_value: 10,
  min_order_amount: 0,
  min_quantity: 1,
  target_scope: 'ENTIRE_TRANSACTION',
  start_date: new Date().toISOString().slice(0, 10),
  end_date: new Date(Date.now() + 30 * 86400000).toISOString().slice(0, 10),
})

async function loadAll() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [custRes, promoRes] = await Promise.all([
      apiRequest<CustomerItem[] | { items: CustomerItem[] }>('/api/v1/customers?limit=50', { method: 'GET' }, auth.accessToken),
      canManagePromo.value
        ? apiRequest<PromotionItem[] | { items: PromotionItem[] }>('/api/v1/promotions?limit=50', { method: 'GET' }, auth.accessToken)
        : Promise.resolve([]),
    ])
    customers.value = unwrapList(custRes)
    promotions.value = unwrapList(promoRes)
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal memuat data promosi dan pelanggan.'
  } finally {
    loading.value = false
  }
}

async function handleCreateCustomer() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest('/api/v1/customers', { method: 'POST', body: JSON.stringify(custForm) }, auth.accessToken)
    actionMsg.value = `Pelanggan "${custForm.name}" berhasil didaftarkan.`
    custForm.code = ''
    custForm.name = ''
    custForm.phone = ''
    custForm.email = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal mendaftarkan pelanggan.'
  }
}

async function handleCreatePromo() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/promotions',
      {
        method: 'POST',
        body: JSON.stringify({
          ...promoForm,
          start_date: `${promoForm.start_date}T00:00:00Z`,
          end_date: `${promoForm.end_date}T23:59:59Z`,
        }),
      },
      auth.accessToken,
    )
    actionMsg.value = `Promosi "${promoForm.name}" berhasil dibuat.`
    promoForm.code = ''
    promoForm.name = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal membuat promosi.'
  }
}

async function handleDeletePromo(p: PromotionItem) {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(`/api/v1/promotions/${p.uuid}`, { method: 'DELETE' }, auth.accessToken)
    actionMsg.value = `Promosi "${p.name}" dinonaktifkan.`
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menghapus promosi.'
  }
}

onMounted(loadAll)
</script>

<template>
  <AppShell page-title="Promosi &amp; Pelanggan" page-subtitle="Kampanye kupon, diskon otomatis, dan daftar member toko.">
    <div v-if="errorMsg" role="alert" class="mb-4 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-4 py-3 text-sm font-medium text-[var(--error-text)]">
      Error: {{ errorMsg }}
    </div>
    <div v-if="actionMsg" role="status" class="mb-4 rounded-md border border-[var(--brand-forest)] bg-[var(--bg-surface)] px-4 py-3 text-sm font-semibold text-[var(--brand-forest)]">
      {{ actionMsg }}
    </div>

    <div role="tablist" aria-label="Menu promosi dan pelanggan" class="mb-5 flex gap-2 rounded-lg border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-1 w-fit">
      <button
        type="button"
        role="tab"
        :aria-selected="activeTab === 'promotions'"
        class="min-h-[44px] rounded-md px-4 py-2 text-sm font-semibold"
        :class="activeTab === 'promotions' ? 'bg-[var(--brand-forest)] text-white' : 'text-[var(--ink-primary)]'"
        @click="activeTab = 'promotions'"
      >
        Promosi ({{ promotions.length }})
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="activeTab === 'customers'"
        class="min-h-[44px] rounded-md px-4 py-2 text-sm font-semibold"
        :class="activeTab === 'customers' ? 'bg-[var(--brand-forest)] text-white' : 'text-[var(--ink-primary)]'"
        @click="activeTab = 'customers'"
      >
        Pelanggan ({{ customers.length }})
      </button>
    </div>

    <div v-if="loading" role="status" class="py-6 text-sm text-[var(--ink-secondary)]">Memuat data...</div>

    <div v-else-if="activeTab === 'promotions'" class="space-y-6">
      <section v-if="canManagePromo" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Buat Kampanye Promosi</h2>
        <form class="mt-3 space-y-3" @submit.prevent="handleCreatePromo">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-4">
            <div>
              <label for="prm-code" class="mb-1 block text-xs font-semibold">Kode Promo / Kupon</label>
              <input id="prm-code" v-model="promoForm.code" required placeholder="GAJIAN10" class="field-input" />
            </div>
            <div>
              <label for="prm-name" class="mb-1 block text-xs font-semibold">Nama Kampanye</label>
              <input id="prm-name" v-model="promoForm.name" required class="field-input" />
            </div>
            <div>
              <label for="prm-type" class="mb-1 block text-xs font-semibold">Tipe Promo</label>
              <select id="prm-type" v-model="promoForm.promo_type" class="field-input">
                <option value="COUPON_CODE">COUPON_CODE</option>
                <option value="AUTOMATIC">AUTOMATIC</option>
              </select>
            </div>
            <div>
              <label for="prm-disc-type" class="mb-1 block text-xs font-semibold">Tipe Diskon</label>
              <select id="prm-disc-type" v-model="promoForm.discount_type" class="field-input">
                <option value="PERCENTAGE">PERCENTAGE (%)</option>
                <option value="FIXED_AMOUNT">FIXED_AMOUNT (Rp)</option>
              </select>
            </div>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-4">
            <div>
              <label for="prm-val" class="mb-1 block text-xs font-semibold">Nilai Diskon</label>
              <input id="prm-val" v-model.number="promoForm.discount_value" type="number" min="1" required class="field-input" />
            </div>
            <div>
              <label for="prm-min" class="mb-1 block text-xs font-semibold">Min. Belanja (Rp)</label>
              <input id="prm-min" v-model.number="promoForm.min_order_amount" type="number" min="0" class="field-input" />
            </div>
            <div>
              <label for="prm-start" class="mb-1 block text-xs font-semibold">Mulai</label>
              <input id="prm-start" v-model="promoForm.start_date" type="date" required class="field-input" />
            </div>
            <div>
              <label for="prm-end" class="mb-1 block text-xs font-semibold">Berakhir</label>
              <input id="prm-end" v-model="promoForm.end_date" type="date" required class="field-input" />
            </div>
          </div>
          <button type="submit" class="min-h-[44px] rounded-md bg-[var(--brand-forest)] px-5 py-2 text-sm font-semibold text-white">
            Simpan Promosi
          </button>
        </form>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Daftar Promosi</h2>
        <p v-if="!promotions.length" class="text-sm text-[var(--ink-secondary)]">Belum ada kampanye promosi.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
                <th scope="col" class="px-3 py-2.5">Kode</th>
                <th scope="col" class="px-3 py-2.5">Nama</th>
                <th scope="col" class="px-3 py-2.5">Diskon</th>
                <th scope="col" class="px-3 py-2.5">Periode</th>
                <th scope="col" class="px-3 py-2.5">Aksi</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in promotions" :key="p.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
                <th scope="row" class="px-3 py-2.5 font-semibold">{{ p.code }}</th>
                <td class="px-3 py-2.5">{{ p.name }} ({{ p.promo_type }})</td>
                <td class="px-3 py-2.5 font-semibold text-[var(--brand-forest)]">
                  {{ p.discount_type === 'PERCENTAGE' ? `${p.discount_value}%` : formatIDR(p.discount_value) }}
                </td>
                <td class="px-3 py-2.5 text-xs text-[var(--ink-secondary)]">
                  {{ formatDate(p.start_date) }} s/d {{ formatDate(p.end_date) }}
                </td>
                <td class="px-3 py-2.5">
                  <button
                    v-if="canManagePromo"
                    type="button"
                    class="min-h-[36px] rounded border border-[var(--error-text)] px-2.5 py-1 text-xs font-semibold text-[var(--error-text)]"
                    @click="handleDeletePromo(p)"
                  >
                    Hapus
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <!-- Customers -->
    <div v-else class="space-y-6">
      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Daftarkan Pelanggan / Member</h2>
        <form class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-5" @submit.prevent="handleCreateCustomer">
          <div>
            <label for="cust-code" class="mb-1 block text-xs font-semibold">Kode</label>
            <input id="cust-code" v-model="custForm.code" required placeholder="CUST-01" class="field-input" />
          </div>
          <div>
            <label for="cust-name" class="mb-1 block text-xs font-semibold">Nama Lengkap</label>
            <input id="cust-name" v-model="custForm.name" required class="field-input" />
          </div>
          <div>
            <label for="cust-phone" class="mb-1 block text-xs font-semibold">No. HP</label>
            <input id="cust-phone" v-model="custForm.phone" type="tel" required class="field-input" />
          </div>
          <div>
            <label for="cust-tier" class="mb-1 block text-xs font-semibold">Tier</label>
            <select id="cust-tier" v-model="custForm.member_tier" class="field-input">
              <option value="REGULAR">REGULAR</option>
              <option value="SILVER">SILVER</option>
              <option value="GOLD">GOLD</option>
              <option value="PLATINUM">PLATINUM</option>
            </select>
          </div>
          <div class="flex items-end">
            <button type="submit" class="min-h-[48px] w-full rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
              Simpan
            </button>
          </div>
        </form>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Daftar Pelanggan</h2>
        <p v-if="!customers.length" class="text-sm text-[var(--ink-secondary)]">Belum ada pelanggan terdaftar.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
                <th scope="col" class="px-3 py-2.5">Kode</th>
                <th scope="col" class="px-3 py-2.5">Nama</th>
                <th scope="col" class="px-3 py-2.5">Telepon</th>
                <th scope="col" class="px-3 py-2.5">Tier</th>
                <th scope="col" class="px-3 py-2.5">Total Belanja</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in customers" :key="c.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
                <th scope="row" class="px-3 py-2.5 font-semibold">{{ c.code }}</th>
                <td class="px-3 py-2.5">{{ c.name }}</td>
                <td class="px-3 py-2.5">{{ c.phone }}</td>
                <td class="px-3 py-2.5">{{ c.member_tier }}</td>
                <td class="px-3 py-2.5 font-semibold">{{ formatIDR(c.total_spend) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppShell>
</template>
