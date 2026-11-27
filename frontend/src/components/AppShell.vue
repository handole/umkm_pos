<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

defineProps<{
  pageTitle: string
  pageSubtitle?: string
}>()

const router = useRouter()
const auth = useAuthStore()

const drawerRef = ref<HTMLDialogElement | null>(null)
const isDrawerOpen = ref(false)

function hasPerm(perm?: string): boolean {
  if (!perm) return true
  if (auth.user?.role === 'SUPERADMIN') return true
  return auth.permissions.includes(perm)
}

const navItems = computed(() =>
  [
    { to: '/', label: 'Ringkasan Operasional' },
    { to: '/pos', label: 'Kasir POS & Shift', perm: 'pos:checkout' },
    { to: '/products', label: 'Katalog Produk', perm: 'product:read' },
    { to: '/inventory', label: 'Stok & Pembelian', perm: 'inventory:read' },
    { to: '/promotions', label: 'Promosi & Pelanggan', perm: 'pos:checkout' },
    { to: '/payroll', label: 'Absensi & Penggajian', perm: 'payroll:read' },
  ].filter((item) => hasPerm(item.perm)),
)

function openDrawer() {
  if (!drawerRef.value) return
  drawerRef.value.showModal()
  isDrawerOpen.value = true
}

function closeDrawer() {
  if (!drawerRef.value) return
  drawerRef.value.close()
  isDrawerOpen.value = false
}

function onDialogClick(event: MouseEvent) {
  if (event.target === drawerRef.value) {
    closeDrawer()
  }
}

async function handleLogout() {
  closeDrawer()
  auth.logout()
  await router.push('/login')
}
</script>

<template>
  <div class="min-h-dvh bg-[var(--bg-paper)] text-[var(--ink-primary)] lg:grid lg:grid-cols-[248px_1fr]">
    <a
      href="#main-content"
      class="visually-hidden fixed top-3 left-3 z-50 rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white"
    >
      Langsung ke konten utama
    </a>

    <!-- Desktop Sidebar -->
    <aside
      aria-label="Sidebar utama"
      class="hidden border-r border-[var(--border-ledger)] bg-[var(--bg-surface)] lg:flex lg:flex-col lg:justify-between"
    >
      <div>
        <div class="border-b border-[var(--border-ledger)] px-5 py-4">
          <RouterLink
            to="/"
            class="inline-flex items-center gap-2.5 text-base font-bold tracking-tight text-[var(--brand-forest)] no-underline"
          >
            <span
              class="inline-flex h-8 w-8 items-center justify-center rounded-md bg-[var(--brand-forest)] text-xs font-extrabold text-white"
              aria-hidden="true"
            >
              UP
            </span>
            <span>UMKM POS</span>
          </RouterLink>
          <p v-if="auth.tenant" class="mt-2 truncate text-xs font-medium text-[var(--ink-secondary)]">
            {{ auth.tenant.name }}
          </p>
        </div>

        <nav aria-label="Navigasi utama" class="p-3">
          <ul class="space-y-1">
            <li v-for="item in navItems" :key="item.to">
              <RouterLink
                :to="item.to"
                class="flex min-h-[44px] items-center rounded-md px-3 py-2 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)]"
                exact-active-class="bg-[var(--brand-forest)] text-white hover:bg-[var(--brand-forest)]"
              >
                {{ item.label }}
              </RouterLink>
            </li>
          </ul>
        </nav>
      </div>

      <div class="border-t border-[var(--border-ledger)] p-4">
        <div v-if="auth.user" class="mb-3">
          <p class="truncate text-sm font-semibold text-[var(--ink-primary)]">
            {{ auth.user.full_name }}
          </p>
          <p class="truncate text-xs text-[var(--ink-secondary)]">
            {{ auth.user.role }}
          </p>
        </div>
        <button
          type="button"
          class="inline-flex min-h-[44px] w-full items-center justify-center rounded-md border border-[var(--border-strong)] bg-[var(--bg-surface)] px-3 py-2 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)]"
          @click="handleLogout"
        >
          Keluar
        </button>
      </div>
    </aside>

    <!-- Mobile Navigation Drawer (<dialog> modal) -->
    <dialog
      id="mobile-nav-drawer"
      ref="drawerRef"
      aria-label="Menu navigasi"
      class="m-0 h-dvh max-h-none w-[min(280px,82dvw)] max-w-none border-r border-[var(--border-ledger)] bg-[var(--bg-surface)] p-0 backdrop:bg-black/45"
      @click="onDialogClick"
      @close="isDrawerOpen = false"
    >
      <div class="flex h-full flex-col justify-between">
        <div>
          <div class="flex items-center justify-between border-b border-[var(--border-ledger)] px-4 py-3.5">
            <span class="text-base font-bold text-[var(--brand-forest)]">UMKM POS</span>
            <button
              type="button"
              autofocus
              aria-label="Tutup menu"
              class="inline-flex min-h-[44px] min-w-[44px] items-center justify-center rounded-md border border-[var(--border-ledger)] px-3 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)]"
              @click="closeDrawer"
            >
              Tutup
            </button>
          </div>

          <nav aria-label="Navigasi mobile" class="p-3">
            <ul class="space-y-1">
              <li v-for="item in navItems" :key="item.to">
                <RouterLink
                  :to="item.to"
                  class="flex min-h-[44px] items-center rounded-md px-3 py-2 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)]"
                  exact-active-class="bg-[var(--brand-forest)] text-white hover:bg-[var(--brand-forest)]"
                  @click="closeDrawer"
                >
                  {{ item.label }}
                </RouterLink>
              </li>
            </ul>
          </nav>
        </div>

        <div class="border-t border-[var(--border-ledger)] p-4">
          <div v-if="auth.user" class="mb-3">
            <p class="truncate text-sm font-semibold text-[var(--ink-primary)]">{{ auth.user.full_name }}</p>
            <p class="truncate text-xs text-[var(--ink-secondary)]">{{ auth.user.role }}</p>
          </div>
          <button
            type="button"
            class="inline-flex min-h-[44px] w-full items-center justify-center rounded-md border border-[var(--border-strong)] bg-[var(--bg-surface)] px-3 py-2 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)]"
            @click="handleLogout"
          >
            Keluar
          </button>
        </div>
      </div>
    </dialog>

    <!-- Main Area -->
    <div class="flex min-w-0 flex-col">
      <header class="border-b border-[var(--border-ledger)] bg-[var(--bg-surface)]">
        <div class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-3.5 sm:px-6">
          <div class="flex items-center gap-3">
            <button
              type="button"
              aria-label="Menu"
              aria-controls="mobile-nav-drawer"
              :aria-expanded="isDrawerOpen"
              class="inline-flex min-h-[44px] items-center rounded-md border border-[var(--border-strong)] bg-[var(--bg-surface)] px-3.5 py-2 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)] lg:hidden"
              @click="openDrawer"
            >
              Menu
            </button>
            <div>
              <h1 class="text-lg font-bold tracking-tight text-[var(--ink-primary)] sm:text-xl">
                {{ pageTitle }}
              </h1>
              <p v-if="pageSubtitle" class="text-xs text-[var(--ink-secondary)] sm:text-sm">
                {{ pageSubtitle }}
              </p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <slot name="header-actions" />
          </div>
        </div>
      </header>

      <main id="main-content" tabindex="-1" class="mx-auto w-full max-w-6xl flex-1 px-4 py-6 sm:px-6">
        <slot />
      </main>
    </div>
  </div>
</template>
