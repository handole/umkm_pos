<script setup lang="ts">
defineProps<{
  heading: string
  subheading: string
}>()
</script>

<template>
  <div class="min-h-dvh bg-[var(--bg-paper)] text-[var(--ink-primary)] flex flex-col">
    <a
      href="#auth-main"
      class="visually-hidden fixed top-3 left-3 z-50 rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white"
    >
      Langsung ke formulir
    </a>

    <header class="border-b border-[var(--border-ledger)] bg-[var(--bg-surface)]">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-3.5 sm:px-6">
        <RouterLink
          to="/login"
          class="inline-flex items-center gap-2.5 text-base font-bold tracking-tight text-[var(--brand-forest)] no-underline"
        >
          <span
            class="inline-flex h-8 w-8 items-center justify-center rounded-md bg-[var(--brand-forest)] text-xs font-extrabold tracking-wider text-white"
            aria-hidden="true"
          >
            UP
          </span>
          <span>UMKM POS</span>
        </RouterLink>

        <nav aria-label="Navigasi autentikasi" class="flex items-center gap-2">
          <RouterLink
            to="/login"
            class="inline-flex min-h-[44px] items-center rounded-md px-3.5 py-2 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)]"
            active-class="bg-[var(--bg-subtle)] text-[var(--brand-forest)]"
          >
            Masuk
          </RouterLink>
          <RouterLink
            to="/register"
            class="inline-flex min-h-[44px] items-center rounded-md px-3.5 py-2 text-sm font-semibold text-[var(--ink-primary)] hover:bg-[var(--bg-subtle)]"
            active-class="bg-[var(--bg-subtle)] text-[var(--brand-forest)]"
          >
            Daftar
          </RouterLink>
        </nav>
      </div>
    </header>

    <main
      id="auth-main"
      tabindex="-1"
      class="mx-auto grid w-full max-w-6xl flex-1 grid-cols-1 gap-8 px-4 py-6 sm:px-6 lg:grid-cols-12 lg:items-start lg:gap-12 lg:py-12"
    >
      <!-- Form column comes first in DOM and on mobile so keyboard never hides submit -->
      <section class="order-1 lg:order-2 lg:col-span-6">
        <div class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5 sm:p-8">
          <div class="mb-6">
            <h1 class="text-2xl font-bold tracking-tight text-[var(--ink-primary)] sm:text-[1.75rem]">
              {{ heading }}
            </h1>
            <p class="mt-1.5 text-sm text-[var(--ink-secondary)] sm:text-base">
              {{ subheading }}
            </p>
          </div>

          <slot />
        </div>
      </section>

      <!-- Editorial / operational ledger context -->
      <aside
        aria-label="Informasi sistem"
        class="order-2 flex flex-col justify-between gap-6 border-t border-[var(--border-ledger)] pt-6 lg:order-1 lg:col-span-6 lg:border-t-0 lg:pt-4"
      >
        <div>
          <p class="text-xs font-bold uppercase tracking-wider text-[var(--accent-terracotta)]">
            Operasional Toko &amp; Backoffice
          </p>
          <h2 class="mt-2 text-xl font-bold leading-snug text-[var(--ink-primary)] sm:text-2xl">
            Satu akses untuk kasir, stok gudang, promosi, dan penggajian staf.
          </h2>
          <p class="mt-3 text-sm leading-relaxed text-[var(--ink-secondary)] sm:text-base">
            Setiap transaksi penjualan langsung terhubung dengan kartu stok multi-lokasi, antrean komisi staf, dan rekapitulasi shift kasir.
          </p>
        </div>

        <dl class="grid grid-cols-1 gap-4 border-t border-[var(--border-ledger)] pt-5 sm:grid-cols-2">
          <div>
            <dt class="text-sm font-semibold text-[var(--ink-primary)]">Kasir &amp; Shift Laci</dt>
            <dd class="mt-1 text-sm text-[var(--ink-secondary)]">
              Mendukung login cepat PIN 6 digit untuk pergantian kasir di tablet maupun desktop.
            </dd>
          </div>
          <div>
            <dt class="text-sm font-semibold text-[var(--ink-primary)]">Hak Akses Peran (RBAC)</dt>
            <dd class="mt-1 text-sm text-[var(--ink-secondary)]">
              Akses dipisahkan sesuai peran: Superadmin, Store Manager, Kasir, Gudang, dan HR/Keuangan.
            </dd>
          </div>
        </dl>
      </aside>
    </main>
  </div>
</template>
