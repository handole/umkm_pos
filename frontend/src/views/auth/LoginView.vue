<script setup lang="ts">
import { nextTick, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../../api/client'
import AuthLayout from '../../components/AuthLayout.vue'
import { useAuthStore } from '../../stores/auth'

type LoginMode = 'password' | 'pin'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const mode = ref<LoginMode>('password')
const showPassword = ref(false)
const showPin = ref(false)
const isSubmitting = ref(false)
const serverError = ref('')

const passwordForm = reactive({
  username: '',
  password: '',
  tenant_code: '',
})

const pinForm = reactive({
  username: '',
  pin: '',
  outlet_id: '',
})

const fieldErrors = reactive<Record<string, string>>({})

function switchMode(nextMode: LoginMode) {
  mode.value = nextMode
  serverError.value = ''
  Object.keys(fieldErrors).forEach((k) => delete fieldErrors[k])
}

function validateField(name: string, el: HTMLInputElement | null) {
  if (!el) return
  if (!el.validity.valid) {
    if (el.validity.valueMissing) {
      fieldErrors[name] = 'Bagian ini wajib diisi.'
    } else if (el.validity.patternMismatch && name === 'pin') {
      fieldErrors[name] = 'PIN harus terdiri dari tepat 6 digit angka.'
    } else {
      fieldErrors[name] = el.validationMessage || 'Format masukan belum sesuai.'
    }
  } else {
    delete fieldErrors[name]
  }
}

function onBlur(event: FocusEvent) {
  const el = event.target as HTMLInputElement
  if (el && el.name) {
    validateField(el.name, el)
  }
}

function onInput(event: Event) {
  const el = event.target as HTMLInputElement
  if (el && el.name && fieldErrors[el.name]) {
    if (el.validity.valid) {
      delete fieldErrors[el.name]
    }
  }
  if (serverError.value) {
    serverError.value = ''
  }
}

async function handleSubmit(event: Event) {
  const form = event.target as HTMLFormElement
  serverError.value = ''

  const inputs = Array.from(form.querySelectorAll<HTMLInputElement>('input'))
  inputs.forEach((input) => {
    if (input.name) validateField(input.name, input)
  })

  if (!form.checkValidity()) {
    const firstInvalid = form.querySelector<HTMLInputElement>(':invalid')
    if (firstInvalid) {
      firstInvalid.focus()
    }
    return
  }

  isSubmitting.value = true
  try {
    if (mode.value === 'password') {
      await auth.login({
        username: passwordForm.username.trim(),
        password: passwordForm.password,
        tenant_code: passwordForm.tenant_code.trim() || undefined,
      })
    } else {
      const parsedOutlet = pinForm.outlet_id.trim() ? Number(pinForm.outlet_id.trim()) : undefined
      await auth.pinLogin({
        username: pinForm.username.trim(),
        pin: pinForm.pin.trim(),
        outlet_id: parsedOutlet,
      })
    }

    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.push(redirect)
  } catch (err) {
    if (err instanceof ApiError) {
      serverError.value = err.message
    } else {
      serverError.value = 'Tidak dapat terhubung ke server. Periksa koneksi atau status backend.'
    }
    await nextTick()
    document.getElementById('login-server-error')?.focus()
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <AuthLayout
    heading="Masuk ke Akun Anda"
    subheading="Gunakan kata sandi akun atau PIN 6 digit untuk akses kasir cepat."
  >
    <!-- Mode Tabs -->
    <div
      role="tablist"
      aria-label="Metode masuk"
      class="mb-6 grid grid-cols-2 gap-2 rounded-lg border border-[var(--border-ledger)] bg-[var(--bg-subtle)] p-1"
    >
      <button
        type="button"
        role="tab"
        :aria-selected="mode === 'password'"
        class="min-h-[44px] rounded-md px-3 py-2 text-sm font-semibold transition-colors"
        :class="
          mode === 'password'
            ? 'bg-[var(--brand-forest)] text-white'
            : 'text-[var(--ink-primary)] hover:bg-[var(--bg-surface)]'
        "
        @click="switchMode('password')"
      >
        Kata Sandi
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="mode === 'pin'"
        class="min-h-[44px] rounded-md px-3 py-2 text-sm font-semibold transition-colors"
        :class="
          mode === 'pin'
            ? 'bg-[var(--brand-forest)] text-white'
            : 'text-[var(--ink-primary)] hover:bg-[var(--bg-surface)]'
        "
        @click="switchMode('pin')"
      >
        PIN Kasir (6 Digit)
      </button>
    </div>

    <!-- Server Error Banner -->
    <div
      v-if="serverError"
      id="login-server-error"
      role="alert"
      tabindex="-1"
      class="mb-5 flex items-start gap-2.5 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-3.5 py-3 text-sm font-medium text-[var(--error-text)]"
    >
      <span class="font-bold" aria-hidden="true">!</span>
      <span>{{ serverError }}</span>
    </div>

    <!-- Password Login Form -->
    <form
      v-if="mode === 'password'"
      novalidate
      aria-label="Formulir masuk dengan kata sandi"
      class="space-y-4"
      @submit.prevent="handleSubmit"
    >
      <div>
        <label for="login-username" class="mb-1.5 block text-sm font-semibold text-[var(--ink-primary)]">
          Username atau Email
        </label>
        <input
          id="login-username"
          v-model="passwordForm.username"
          name="username"
          type="text"
          autocomplete="username"
          autocorrect="off"
          autocapitalize="none"
          spellcheck="false"
          enterkeyhint="next"
          required
          :aria-invalid="Boolean(fieldErrors.username)"
          :aria-describedby="fieldErrors.username ? 'login-username-error' : undefined"
          class="field-input"
          @blur="onBlur"
          @input="onInput"
        />
        <p
          v-if="fieldErrors.username"
          id="login-username-error"
          class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
        >
          Error: {{ fieldErrors.username }}
        </p>
      </div>

      <div>
        <div class="mb-1.5 flex items-center justify-between gap-2">
          <label for="login-password" class="block text-sm font-semibold text-[var(--ink-primary)]">
            Kata Sandi
          </label>
        </div>
        <div class="relative">
          <input
            id="login-password"
            v-model="passwordForm.password"
            name="password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="current-password"
            enterkeyhint="done"
            required
            :aria-invalid="Boolean(fieldErrors.password)"
            :aria-describedby="fieldErrors.password ? 'login-password-error' : undefined"
            class="field-input pr-24"
            @blur="onBlur"
            @input="onInput"
          />
          <button
            type="button"
            :aria-pressed="showPassword"
            :aria-label="showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'"
            class="absolute top-1/2 right-1.5 -translate-y-1/2 rounded px-2.5 py-2 text-xs font-semibold text-[var(--brand-forest)] hover:bg-[var(--bg-subtle)]"
            @click="showPassword = !showPassword"
          >
            {{ showPassword ? 'Sembunyikan' : 'Tampilkan' }}
          </button>
        </div>
        <p
          v-if="fieldErrors.password"
          id="login-password-error"
          class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
        >
          Error: {{ fieldErrors.password }}
        </p>
      </div>

      <details class="rounded-md border border-[var(--border-ledger)] bg-[var(--bg-paper)] px-3.5 py-2.5">
        <summary class="min-h-[36px] flex items-center text-sm font-semibold text-[var(--ink-secondary)]">
          Opsi Multi-Tenant (Opsional)
        </summary>
        <div class="mt-3 pb-1">
          <label for="login-tenant" class="mb-1.5 block text-sm font-semibold text-[var(--ink-primary)]">
            Kode Tenant / Usaha
          </label>
          <span id="login-tenant-hint" class="mb-1.5 block text-xs text-[var(--ink-secondary)]">
            Kosongkan jika menggunakan mode single-tenant (default).
          </span>
          <input
            id="login-tenant"
            v-model="passwordForm.tenant_code"
            name="tenant_code"
            type="text"
            autocorrect="off"
            autocapitalize="none"
            spellcheck="false"
            aria-describedby="login-tenant-hint"
            class="field-input"
          />
        </div>
      </details>

      <button
        type="submit"
        :disabled="isSubmitting"
        class="mt-2 inline-flex min-h-[48px] w-full items-center justify-center rounded-md bg-[var(--brand-forest)] px-5 py-3 text-base font-semibold text-white transition-colors hover:bg-[var(--brand-forest-hover)] disabled:opacity-65"
      >
        {{ isSubmitting ? 'Memeriksa kredensial...' : 'Masuk ke Sistem' }}
      </button>
    </form>

    <!-- Quick PIN Login Form -->
    <form
      v-else
      novalidate
      aria-label="Formulir masuk dengan PIN kasir"
      class="space-y-4"
      @submit.prevent="handleSubmit"
    >
      <div>
        <label for="pin-username" class="mb-1.5 block text-sm font-semibold text-[var(--ink-primary)]">
          Username Staf
        </label>
        <input
          id="pin-username"
          v-model="pinForm.username"
          name="username"
          type="text"
          autocomplete="username"
          autocorrect="off"
          autocapitalize="none"
          spellcheck="false"
          enterkeyhint="next"
          required
          :aria-invalid="Boolean(fieldErrors.username)"
          :aria-describedby="fieldErrors.username ? 'pin-username-error' : undefined"
          class="field-input"
          @blur="onBlur"
          @input="onInput"
        />
        <p
          v-if="fieldErrors.username"
          id="pin-username-error"
          class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
        >
          Error: {{ fieldErrors.username }}
        </p>
      </div>

      <div>
        <label for="pin-code" class="mb-1 block text-sm font-semibold text-[var(--ink-primary)]">
          PIN 6 Digit
        </label>
        <span id="pin-code-hint" class="mb-1.5 block text-xs text-[var(--ink-secondary)]">
          Masukkan 6 angka PIN kasir atau kiosk absensi.
        </span>
        <div class="relative">
          <input
            id="pin-code"
            v-model="pinForm.pin"
            name="pin"
            :type="showPin ? 'text' : 'password'"
            inputmode="numeric"
            pattern="[0-9]{6}"
            maxlength="6"
            autocomplete="off"
            enterkeyhint="done"
            required
            :aria-invalid="Boolean(fieldErrors.pin)"
            :aria-describedby="fieldErrors.pin ? 'pin-code-hint pin-code-error' : 'pin-code-hint'"
            class="field-input pr-24 tracking-widest"
            @blur="onBlur"
            @input="onInput"
          />
          <button
            type="button"
            :aria-pressed="showPin"
            :aria-label="showPin ? 'Sembunyikan PIN' : 'Tampilkan PIN'"
            class="absolute top-1/2 right-1.5 -translate-y-1/2 rounded px-2.5 py-2 text-xs font-semibold text-[var(--brand-forest)] hover:bg-[var(--bg-subtle)]"
            @click="showPin = !showPin"
          >
            {{ showPin ? 'Sembunyikan' : 'Tampilkan' }}
          </button>
        </div>
        <p
          v-if="fieldErrors.pin"
          id="pin-code-error"
          class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
        >
          Error: {{ fieldErrors.pin }}
        </p>
      </div>

      <div>
        <label for="pin-outlet" class="mb-1.5 block text-sm font-semibold text-[var(--ink-primary)]">
          ID Outlet (Opsional)
        </label>
        <input
          id="pin-outlet"
          v-model="pinForm.outlet_id"
          name="outlet_id"
          type="text"
          inputmode="numeric"
          class="field-input"
        />
      </div>

      <button
        type="submit"
        :disabled="isSubmitting"
        class="mt-2 inline-flex min-h-[48px] w-full items-center justify-center rounded-md bg-[var(--brand-forest)] px-5 py-3 text-base font-semibold text-white transition-colors hover:bg-[var(--brand-forest-hover)] disabled:opacity-65"
      >
        {{ isSubmitting ? 'Memverifikasi PIN...' : 'Masuk dengan PIN' }}
      </button>
    </form>

    <div class="mt-6 border-t border-[var(--border-ledger)] pt-5 text-sm text-[var(--ink-secondary)]">
      Belum memiliki akun staf atau pemilik toko?
      <RouterLink to="/register" class="font-semibold text-[var(--accent-terracotta)] underline underline-offset-4">
        Daftarkan akun baru
      </RouterLink>
    </div>
  </AuthLayout>
</template>
