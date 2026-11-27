<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const isLoading = ref(false)
const loadError = ref('')

async function verifySession() {
  isLoading.value = true
  loadError.value = ''
  try {
    await auth.fetchMe()
  } catch {
    loadError.value = 'Gagal memverifikasi sesi ke server. Periksa koneksi backend Anda.'
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  verifySession()
})

async function handleLogout() {
  auth.logout()
  await router.push('/login')
}
</script>

<template>
  <div class="min-h-dvh bg-[var(--bg-paper)] text-[var(--ink-primary)]">
    <header class="border-b border-[var(--border-ledger)] bg-[var(--bg-surface)]">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-3.5 sm:px-6">
        <div class="flex items-center gap-2.5 text-base font-bold text-[var(--brand-forest)]">
          <span
            class="inline-flex h-8 w-8 items-center justify-center rounded-md bg-[var(--brand-forest)] text-xs font-extrabold text-white"
            aria-hidden="true"
          >
            UP
          </span>
          <span>UMKM POS</span>
        </div>

        <button
          type="button"
          class="inline-flex min-h-[44px] items-center rounded-md border border-[var(--border-strong)] bg-[var(--bg-surface)] px-4 py-2 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)]"
          @click="handleLogout"
        >
          Keluar
        </button>
      </div>
    </header>

    <main class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-[var(--ink-primary)]">
          Ringkasan Sesi Aktif
        </h1>
        <p class="mt-1 text-sm text-[var(--ink-secondary)]">
          Anda berhasil masuk. Modul operasional berikutnya (Produk, Stok, Kasir, Promosi, dan Payroll) akan ditambahkan di sini.
        </p>
      </div>

      <div
        v-if="isLoading"
        role="status"
        class="mb-6 rounded-md border border-[var(--border-ledger)] bg-[var(--bg-surface)] px-4 py-3 text-sm text-[var(--ink-secondary)]"
      >
        Memverifikasi sesi aktif dengan server...
      </div>

      <div
        v-else-if="loadError"
        role="alert"
        class="mb-6 flex flex-wrap items-center justify-between gap-3 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-4 py-3 text-sm text-[var(--error-text)]"
      >
        <span>{{ loadError }}</span>
        <button
          type="button"
          class="min-h-[44px] rounded border border-[var(--error-text)] px-3 py-1.5 text-xs font-semibold"
          @click="verifySession"
        >
          Coba Lagi
        </button>
      </div>

      <div v-if="auth.user" class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
          <h2 class="text-base font-bold text-[var(--ink-primary)]">Identitas Pengguna</h2>
          <dl class="mt-4 space-y-2.5 text-sm">
            <div class="flex justify-between gap-4 border-b border-[var(--border-ledger)] pb-2">
              <dt class="text-[var(--ink-secondary)]">Nama</dt>
              <dd class="font-semibold">{{ auth.user.full_name }}</dd>
            </div>
            <div class="flex justify-between gap-4 border-b border-[var(--border-ledger)] pb-2">
              <dt class="text-[var(--ink-secondary)]">Username</dt>
              <dd class="font-semibold">{{ auth.user.username }}</dd>
            </div>
            <div class="flex justify-between gap-4 border-b border-[var(--border-ledger)] pb-2">
              <dt class="text-[var(--ink-secondary)]">Peran</dt>
              <dd class="font-semibold text-[var(--brand-forest)]">{{ auth.user.role }}</dd>
            </div>
            <div v-if="auth.tenant" class="flex justify-between gap-4">
              <dt class="text-[var(--ink-secondary)]">Usaha / Tenant</dt>
              <dd class="font-semibold">{{ auth.tenant.name }} ({{ auth.tenant.code }})</dd>
            </div>
          </dl>
        </section>

        <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
          <h2 class="text-base font-bold text-[var(--ink-primary)]">Outlet &amp; Hak Akses</h2>
          <div class="mt-4">
            <h3 class="text-xs font-bold uppercase tracking-wide text-[var(--ink-secondary)]">Outlet Terdaftar</h3>
            <p v-if="!auth.outlets.length" class="mt-1.5 text-sm text-[var(--ink-secondary)]">
              Belum ada outlet terhubung pada akun ini.
            </p>
            <ul v-else class="mt-2 space-y-1.5 text-sm">
              <li v-for="o in auth.outlets" :key="o.id" class="flex items-center justify-between">
                <span>{{ o.name }} ({{ o.code }})</span>
                <span v-if="o.is_main_branch" class="text-xs font-semibold text-[var(--brand-forest)]">Utama</span>
              </li>
            </ul>
          </div>

          <div class="mt-5 border-t border-[var(--border-ledger)] pt-4">
            <h3 class="text-xs font-bold uppercase tracking-wide text-[var(--ink-secondary)]">
              Permission ({{ auth.permissions.length }})
            </h3>
            <p v-if="!auth.permissions.length" class="mt-1.5 text-sm text-[var(--ink-secondary)]">
              Tidak ada permission yang tercatat.
            </p>
            <ul v-else class="mt-2 flex flex-wrap gap-1.5">
              <li
                v-for="perm in auth.permissions"
                :key="perm"
                class="rounded border border-[var(--border-ledger)] bg-[var(--bg-paper)] px-2 py-1 text-xs font-medium text-[var(--ink-primary)]"
              >
                {{ perm }}
              </li>
            </ul>
          </div>
        </section>
      </div>
    </main>
  </div>
</template>
