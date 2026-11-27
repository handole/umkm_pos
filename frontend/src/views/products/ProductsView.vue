<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError, apiRequest, unwrapList } from '../../api/client'
import AppShell from '../../components/AppShell.vue'
import { useAuthStore } from '../../stores/auth'
import { formatIDR } from '../../utils/format'

interface CategoryItem {
  id: number
  uuid: string
  name: string
  slug: string
  is_active: boolean
}

interface ProductUOM {
  id: number
  uuid: string
  uom_name: string
  conversion_rate: number
  sell_price: number
  barcode?: string
  is_base_uom: boolean
}

interface ProductVariant {
  id: number
  uuid: string
  sku: string
  barcode?: string
  variant_name: string
  base_cost_price: number
  base_sell_price: number
  min_stock_alert: number
  uoms?: ProductUOM[]
}

interface ProductItem {
  id: number
  uuid: string
  name: string
  description?: string
  is_active: boolean
  category?: CategoryItem
  variants?: ProductVariant[]
}

const auth = useAuthStore()

function hasPerm(perm: string): boolean {
  if (auth.user?.role === 'SUPERADMIN') return true
  return auth.permissions.includes(perm)
}

const canCreate = computed(() => hasPerm('product:create'))
const canDelete = computed(() => hasPerm('product:delete'))

const activeTab = ref<'products' | 'categories'>('products')
const search = ref('')
const loading = ref(false)
const errorMsg = ref('')
const actionMsg = ref('')

const products = ref<ProductItem[]>([])
const categories = ref<CategoryItem[]>([])

const showCreateProduct = ref(false)
const submittingProduct = ref(false)
const productForm = reactive({
  name: '',
  category_id: '',
  description: '',
  sku: '',
  barcode: '',
  variant_name: 'Standard',
  base_cost_price: 0,
  base_sell_price: 0,
  min_stock_alert: 5,
  uom_name: 'Pcs',
})

const categoryName = ref('')
const submittingCategory = ref(false)

const barcodeInput = ref('')
const scanResult = ref<string>('')
const scanError = ref('')

async function loadData() {
  loading.value = true
  errorMsg.value = ''
  try {
    const qs = new URLSearchParams({ page: '1', limit: '50' })
    if (search.value.trim()) qs.set('search', search.value.trim())
    const [prodRes, catRes] = await Promise.all([
      apiRequest<ProductItem[] | { items: ProductItem[] }>(
        `/api/v1/products?${qs.toString()}`,
        { method: 'GET' },
        auth.accessToken,
      ),
      apiRequest<CategoryItem[]>('/api/v1/categories', { method: 'GET' }, auth.accessToken),
    ])
    products.value = unwrapList(prodRes)
    categories.value = catRes || []
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal memuat katalog produk.'
  } finally {
    loading.value = false
  }
}

async function handleCreateProduct() {
  submittingProduct.value = true
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/products',
      {
        method: 'POST',
        body: JSON.stringify({
          name: productForm.name.trim(),
          category_id: productForm.category_id ? Number(productForm.category_id) : undefined,
          description: productForm.description.trim() || undefined,
          variants: [
            {
              sku: productForm.sku.trim(),
              barcode: productForm.barcode.trim() || undefined,
              variant_name: productForm.variant_name.trim() || 'Standard',
              base_cost_price: Number(productForm.base_cost_price),
              base_sell_price: Number(productForm.base_sell_price),
              min_stock_alert: Number(productForm.min_stock_alert),
              uoms: [
                {
                  uom_name: productForm.uom_name.trim() || 'Pcs',
                  conversion_rate: 1,
                  sell_price: Number(productForm.base_sell_price),
                  barcode: productForm.barcode.trim() || undefined,
                  is_base_uom: true,
                },
              ],
            },
          ],
        }),
      },
      auth.accessToken,
    )
    actionMsg.value = `Produk "${productForm.name}" berhasil ditambahkan.`
    showCreateProduct.value = false
    productForm.name = ''
    productForm.sku = ''
    productForm.barcode = ''
    productForm.description = ''
    productForm.base_cost_price = 0
    productForm.base_sell_price = 0
    await loadData()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menyimpan produk.'
  } finally {
    submittingProduct.value = false
  }
}

async function handleDeleteProduct(item: ProductItem) {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(`/api/v1/products/${item.uuid}`, { method: 'DELETE' }, auth.accessToken)
    actionMsg.value = `Produk "${item.name}" dinonaktifkan.`
    await loadData()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menghapus produk.'
  }
}

async function handleCreateCategory() {
  if (!categoryName.value.trim()) return
  submittingCategory.value = true
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/categories',
      { method: 'POST', body: JSON.stringify({ name: categoryName.value.trim() }) },
      auth.accessToken,
    )
    actionMsg.value = `Kategori "${categoryName.value.trim()}" berhasil dibuat.`
    categoryName.value = ''
    await loadData()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal membuat kategori.'
  } finally {
    submittingCategory.value = false
  }
}

async function handleDeleteCategory(cat: CategoryItem) {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(`/api/v1/categories/${cat.uuid}`, { method: 'DELETE' }, auth.accessToken)
    actionMsg.value = `Kategori "${cat.name}" dihapus.`
    await loadData()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal menghapus kategori.'
  }
}

async function handleScanBarcode() {
  if (!barcodeInput.value.trim()) return
  scanResult.value = ''
  scanError.value = ''
  try {
    const res = await apiRequest<{
      product: { name: string }
      variant: { sku: string; variant_name: string }
      scan_type: string
      calculated_price: number
      base_unit_multiplier: number
    }>(
      `/api/v1/products/barcode/${encodeURIComponent(barcodeInput.value.trim())}`,
      { method: 'GET' },
      auth.accessToken,
    )
    scanResult.value = `${res.product.name} (${res.variant.sku}) | Harga: ${formatIDR(res.calculated_price)} | Isi: ${res.base_unit_multiplier} unit`
  } catch (err) {
    scanError.value = err instanceof ApiError ? err.message : 'Barcode tidak ditemukan.'
  }
}

onMounted(loadData)
</script>

<template>
  <AppShell page-title="Katalog Produk &amp; Kategori" page-subtitle="Master SKU, harga beli/jual, satuan bertingkat, dan uji pemindaian barcode.">
    <template #header-actions>
      <button
        v-if="canCreate"
        type="button"
        class="inline-flex min-h-[44px] items-center rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white hover:bg-[var(--brand-forest-hover)]"
        @click="showCreateProduct = !showCreateProduct"
      >
        {{ showCreateProduct ? 'Tutup Form Produk' : 'Tambah Produk' }}
      </button>
    </template>

    <div v-if="errorMsg" role="alert" class="mb-4 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-4 py-3 text-sm font-medium text-[var(--error-text)]">
      Error: {{ errorMsg }}
    </div>
    <div v-if="actionMsg" role="status" class="mb-4 rounded-md border border-[var(--brand-forest)] bg-[var(--bg-surface)] px-4 py-3 text-sm font-semibold text-[var(--brand-forest)]">
      {{ actionMsg }}
    </div>

    <!-- Create Product Panel -->
    <section v-if="showCreateProduct && canCreate" class="mb-6 rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
      <h2 class="text-base font-bold text-[var(--ink-primary)]">Tambah Produk Baru</h2>
      <form class="mt-4 space-y-4" @submit.prevent="handleCreateProduct">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label for="prod-name" class="mb-1 block text-sm font-semibold">Nama Produk</label>
            <input id="prod-name" v-model="productForm.name" required class="field-input" />
          </div>
          <div>
            <label for="prod-cat" class="mb-1 block text-sm font-semibold">Kategori</label>
            <select id="prod-cat" v-model="productForm.category_id" class="field-input">
              <option value="">Tanpa Kategori</option>
              <option v-for="c in categories" :key="c.id" :value="String(c.id)">{{ c.name }}</option>
            </select>
          </div>
          <div>
            <label for="prod-sku" class="mb-1 block text-sm font-semibold">SKU Varian</label>
            <input id="prod-sku" v-model="productForm.sku" required class="field-input" />
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-4">
          <div>
            <label for="prod-variant" class="mb-1 block text-sm font-semibold">Nama Varian</label>
            <input id="prod-variant" v-model="productForm.variant_name" required class="field-input" />
          </div>
          <div>
            <label for="prod-barcode" class="mb-1 block text-sm font-semibold">Barcode (Opsional)</label>
            <input id="prod-barcode" v-model="productForm.barcode" class="field-input" />
          </div>
          <div>
            <label for="prod-cost" class="mb-1 block text-sm font-semibold">Harga Beli (HPP)</label>
            <input id="prod-cost" v-model.number="productForm.base_cost_price" type="number" min="0" required class="field-input" />
          </div>
          <div>
            <label for="prod-sell" class="mb-1 block text-sm font-semibold">Harga Jual</label>
            <input id="prod-sell" v-model.number="productForm.base_sell_price" type="number" min="0" required class="field-input" />
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label for="prod-uom" class="mb-1 block text-sm font-semibold">Satuan Dasar</label>
            <input id="prod-uom" v-model="productForm.uom_name" required class="field-input" />
          </div>
          <div>
            <label for="prod-alert" class="mb-1 block text-sm font-semibold">Batas Minimum Stok</label>
            <input id="prod-alert" v-model.number="productForm.min_stock_alert" type="number" min="0" class="field-input" />
          </div>
          <div>
            <label for="prod-desc" class="mb-1 block text-sm font-semibold">Catatan / Deskripsi</label>
            <input id="prod-desc" v-model="productForm.description" class="field-input" />
          </div>
        </div>

        <button
          type="submit"
          :disabled="submittingProduct"
          class="inline-flex min-h-[48px] items-center rounded-md bg-[var(--brand-forest)] px-5 py-2.5 text-sm font-semibold text-white hover:bg-[var(--brand-forest-hover)] disabled:opacity-65"
        >
          {{ submittingProduct ? 'Menyimpan...' : 'Simpan Produk' }}
        </button>
      </form>
    </section>

    <!-- Tabs & Barcode Quick Lookup -->
    <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
      <div role="tablist" aria-label="Tampilan katalog" class="flex gap-2 rounded-lg border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-1">
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'products'"
          class="min-h-[44px] rounded-md px-4 py-2 text-sm font-semibold"
          :class="activeTab === 'products' ? 'bg-[var(--brand-forest)] text-white' : 'text-[var(--ink-primary)]'"
          @click="activeTab = 'products'"
        >
          Daftar Produk ({{ products.length }})
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'categories'"
          class="min-h-[44px] rounded-md px-4 py-2 text-sm font-semibold"
          :class="activeTab === 'categories' ? 'bg-[var(--brand-forest)] text-white' : 'text-[var(--ink-primary)]'"
          @click="activeTab = 'categories'"
        >
          Kategori ({{ categories.length }})
        </button>
      </div>

      <search>
        <form class="flex flex-wrap items-center gap-2" @submit.prevent="handleScanBarcode">
          <label for="scan-barcode" class="visually-hidden">Cek Barcode</label>
          <input
            id="scan-barcode"
            v-model="barcodeInput"
            type="search"
            placeholder="Scan / ketik barcode..."
            class="min-h-[44px] rounded-md border border-[var(--border-strong)] bg-[var(--bg-surface)] px-3 py-2 text-sm"
          />
          <button
            type="submit"
            class="min-h-[44px] rounded-md border border-[var(--border-strong)] bg-[var(--bg-subtle)] px-3.5 py-2 text-sm font-semibold hover:bg-[var(--border-ledger)]"
          >
            Cek Barcode
          </button>
        </form>
      </search>
    </div>

    <p v-if="scanResult" role="status" class="mb-4 rounded-md border border-[var(--brand-forest)] bg-[var(--bg-surface)] px-4 py-2.5 text-sm font-semibold text-[var(--brand-forest)]">
      Hasil Scan: {{ scanResult }}
    </p>
    <p v-if="scanError" role="alert" class="mb-4 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-4 py-2.5 text-sm font-medium text-[var(--error-text)]">
      Error: {{ scanError }}
    </p>

    <!-- Products List -->
    <section v-if="activeTab === 'products'" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
      <search class="mb-4">
        <form class="flex flex-wrap gap-2" @submit.prevent="loadData">
          <label for="search-products" class="visually-hidden">Cari produk</label>
          <input
            id="search-products"
            v-model="search"
            type="search"
            placeholder="Cari nama produk atau SKU..."
            class="min-h-[44px] flex-1 rounded-md border border-[var(--border-strong)] bg-[var(--bg-surface)] px-3 py-2 text-sm"
          />
          <button type="submit" class="min-h-[44px] rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
            Cari
          </button>
        </form>
      </search>

      <div v-if="loading" role="status" class="py-6 text-sm text-[var(--ink-secondary)]">Memuat katalog produk...</div>
      <div v-else-if="!products.length" class="py-6 text-sm text-[var(--ink-secondary)]">
        Belum ada produk terdaftar. Gunakan tombol Tambah Produk untuk membuat master produk pertama.
      </div>
      <div v-else tabindex="0" role="region" aria-label="Tabel katalog produk" class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
              <th scope="col" class="px-3 py-2.5">Produk</th>
              <th scope="col" class="px-3 py-2.5">Kategori</th>
              <th scope="col" class="px-3 py-2.5">SKU &amp; Varian</th>
              <th scope="col" class="px-3 py-2.5">HPP</th>
              <th scope="col" class="px-3 py-2.5">Harga Jual</th>
              <th scope="col" class="px-3 py-2.5">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in products" :key="p.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
              <th scope="row" class="px-3 py-3 font-semibold">{{ p.name }}</th>
              <td class="px-3 py-3 text-[var(--ink-secondary)]">{{ p.category?.name || '-' }}</td>
              <td class="px-3 py-3">
                <span v-if="p.variants?.length">{{ p.variants[0]?.sku }} ({{ p.variants[0]?.variant_name }})</span>
                <span v-else>-</span>
              </td>
              <td class="px-3 py-3">{{ p.variants?.length ? formatIDR(p.variants[0]?.base_cost_price) : '-' }}</td>
              <td class="px-3 py-3 font-semibold text-[var(--brand-forest)]">
                {{ p.variants?.length ? formatIDR(p.variants[0]?.base_sell_price) : '-' }}
              </td>
              <td class="px-3 py-3">
                <button
                  v-if="canDelete"
                  type="button"
                  class="min-h-[36px] rounded border border-[var(--error-text)] px-2.5 py-1 text-xs font-semibold text-[var(--error-text)] hover:bg-[var(--error-bg)]"
                  @click="handleDeleteProduct(p)"
                >
                  Hapus
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Categories Tab -->
    <section v-else class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
      <form v-if="canCreate" class="mb-5 flex flex-wrap gap-2" @submit.prevent="handleCreateCategory">
        <label for="new-cat-name" class="visually-hidden">Nama kategori baru</label>
        <input
          id="new-cat-name"
          v-model="categoryName"
          type="text"
          required
          placeholder="Nama kategori baru..."
          class="min-h-[44px] flex-1 rounded-md border border-[var(--border-strong)] bg-[var(--bg-surface)] px-3 py-2 text-sm"
        />
        <button
          type="submit"
          :disabled="submittingCategory"
          class="min-h-[44px] rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white"
        >
          {{ submittingCategory ? 'Menyimpan...' : 'Tambah Kategori' }}
        </button>
      </form>

      <p v-if="!categories.length" class="text-sm text-[var(--ink-secondary)]">Belum ada kategori produk.</p>
      <ul v-else class="divide-y divide-[var(--border-ledger)] text-sm">
        <li v-for="cat in categories" :key="cat.uuid" class="flex items-center justify-between gap-3 py-2.5">
          <div>
            <span class="font-semibold">{{ cat.name }}</span>
            <span class="ml-2 text-xs text-[var(--ink-secondary)]">/{{ cat.slug }}</span>
          </div>
          <button
            v-if="canDelete"
            type="button"
            class="min-h-[36px] rounded border border-[var(--error-text)] px-2.5 py-1 text-xs font-semibold text-[var(--error-text)] hover:bg-[var(--error-bg)]"
            @click="handleDeleteCategory(cat)"
          >
            Hapus
          </button>
        </li>
      </ul>
    </section>
  </AppShell>
</template>
