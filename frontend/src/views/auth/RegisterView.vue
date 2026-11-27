<script setup lang="ts">
import { nextTick, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError } from '../../api/client'
import AuthLayout from '../../components/AuthLayout.vue'
import { useAuthStore } from '../../stores/auth'
import type { RoleName } from '../../types/auth'

const router = useRouter()
const auth = useAuthStore()

const showPassword = ref(false)
const isSubmitting = ref(false)
const serverError = ref('')

const formState = reactive({
  full_name: '',
  username: '',
  email: '',
  phone: '',
  password: '',
  role_name: 'SUPERADMIN' as RoleName,
  pin: '',
  tenant_code: '',
  tenant_name: '',
})

const roleOptions: Array<{ value: RoleName; label: string }> = [
  { value: 'SUPERADMIN', label: 'Pemilik Usaha / Superadmin' },
  { value: 'STORE_MANAGER', label: 'Store Manager / Supervisor' },
  { value: 'CASHIER', label: 'Kasir (POS)' },
  { value: 'WAREHOUSE', label: 'Staf Gudang' },
  { value: 'HR_FINANCE', label: 'HR & Keuangan' },
]

const fieldErrors = reactive<Record<string, string>>({})

function validateField(name: string, el: HTMLInputElement | HTMLSelectElement | null) {
  if (!el) return
  if (!el.validity.valid) {
    if (el.validity.valueMissing) {
      fieldErrors[name] = 'Bagian ini wajib diisi.'
    } else if (el.validity.tooShort) {
      fieldErrors[name] = `Minimal ${el.getAttribute('minlength')} karakter.`
    } else if (el.validity.typeMismatch && name === 'email') {
      fieldErrors[name] = 'Masukkan alamat email yang valid, misalnya nama@toko.id.'
    } else if (el.validity.patternMismatch && name === 'pin') {
      fieldErrors[name] = 'PIN opsional harus tepat 6 digit angka.'
    } else {
      fieldErrors[name] = el.validationMessage || 'Format masukan belum sesuai.'
    }
  } else {
    delete fieldErrors[name]
  }
}

function onBlur(event: FocusEvent) {
  const el = event.target as HTMLInputElement | HTMLSelectElement
  if (el && el.name) {
    validateField(el.name, el)
  }
}

function onInput(event: Event) {
  const el = event.target as HTMLInputElement | HTMLSelectElement
  if (el && el.name && fieldErrors[el.name] && el.validity.valid) {
    delete fieldErrors[el.name]
  }
  if (serverError.value) {
    serverError.value = ''
  }
}

async function handleSubmit(event: Event) {
  const form = event.target as HTMLFormElement
  serverError.value = ''

  const controls = Array.from(form.querySelectorAll<HTMLInputElement | HTMLSelectElement>('input, select'))
  controls.forEach((ctrl) => {
    if (ctrl.name) validateField(ctrl.name, ctrl)
  })

  if (!form.checkValidity()) {
    const firstInvalid = form.querySelector<HTMLElement>(':invalid')
    if (firstInvalid) {
      firstInvalid.focus()
    }
    return
  }

  isSubmitting.value = true
  try {
    await auth.register({
      full_name: formState.full_name.trim(),
      username: formState.username.trim(),
      email: formState.email.trim() || undefined,
      phone: formState.phone.trim() || undefined,
      password: formState.password,
      role_name: formState.role_name,
      pin: formState.pin.trim() || undefined,
      tenant_code: formState.tenant_code.trim() || undefined,
      tenant_name: formState.tenant_name.trim() || undefined,
    })
    await router.push('/')
  } catch (err) {
    if (err instanceof ApiError) {
      serverError.value = err.message
    } else {
      serverError.value = 'Tidak dapat terhubung ke server. Periksa koneksi atau status backend.'
    }
    await nextTick()
    document.getElementById('register-server-error')?.focus()
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <AuthLayout
    heading="Buat Akun Baru"
    subheading="Daftarkan akun pemilik usaha atau staf untuk mulai menggunakan sistem."
  >
    <div
      v-if="serverError"
      id="register-server-error"
      role="alert"
      tabindex="-1"
      class="mb-5 flex items-start gap-2.5 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-3.5 py-3 text-sm font-medium text-[var(--error-text)]"
    >
      <span class="font-bold" aria-hidden="true">!</span>
      <span>{{ serverError }}</span>
    </div>

    <form
      novalidate
      aria-label="Formulir pendaftaran akun"
      class="space-y-4"
      @submit.prevent="handleSubmit"
    >
      <div>
        <label for="reg-fullname" class="mb-1.5 block text-sm font-semibold text-[var(--ink-primary)]">
          Nama Lengkap
        </label>
        <input
          id="reg-fullname"
          v-model="formState.full_name"
          name="full_name"
          type="text"
          autocomplete="name"
          enterkeyhint="next"
          required
          :aria-invalid="Boolean(fieldErrors.full_name)"
          :aria-describedby="fieldErrors.full_name ? 'reg-fullname-error' : undefined"
          class="field-input"
          @blur="onBlur"
          @input="onInput"
        />
        <p
          v-if="fieldErrors.full_name"
          id="reg-fullname-error"
          class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
        >
          Error: {{ fieldErrors.full_name }}
        </p>
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label for="reg-username" class="mb-1 block text-sm font-semibold text-[var(--ink-primary)]">
            Username
          </label>
          <span id="reg-username-hint" class="mb-1.5 block text-xs text-[var(--ink-secondary)]">
            Minimal 3 karakter, tanpa spasi.
          </span>
          <input
            id="reg-username"
            v-model="formState.username"
            name="username"
            type="text"
            autocomplete="username"
            autocorrect="off"
            autocapitalize="none"
            spellcheck="false"
            minlength="3"
            maxlength="50"
            enterkeyhint="next"
            required
            :aria-invalid="Boolean(fieldErrors.username)"
            :aria-describedby="fieldErrors.username ? 'reg-username-hint reg-username-error' : 'reg-username-hint'"
            class="field-input"
            @blur="onBlur"
            @input="onInput"
          />
          <p
            v-if="fieldErrors.username"
            id="reg-username-error"
            class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
          >
            Error: {{ fieldErrors.username }}
          </p>
        </div>

        <div>
          <label for="reg-role" class="mb-1 block text-sm font-semibold text-[var(--ink-primary)]">
            Peran Akun
          </label>
          <span id="reg-role-hint" class="mb-1.5 block text-xs text-[var(--ink-secondary)]">
            Menentukan izin modul di sistem.
          </span>
          <select
            id="reg-role"
            v-model="formState.role_name"
            name="role_name"
            required
            aria-describedby="reg-role-hint"
            class="field-input"
          >
            <option v-for="r in roleOptions" :key="r.value" :value="r.value">
              {{ r.label }}
            </option>
          </select>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label for="reg-email" class="mb-1.5 block text-sm font-semibold text-[var(--ink-primary)]">
            Email (Opsional)
          </label>
          <input
            id="reg-email"
            v-model="formState.email"
            name="email"
            type="email"
            autocomplete="email"
            spellcheck="false"
            enterkeyhint="next"
            :aria-invalid="Boolean(fieldErrors.email)"
            :aria-describedby="fieldErrors.email ? 'reg-email-error' : undefined"
            class="field-input"
            @blur="onBlur"
            @input="onInput"
          />
          <p
            v-if="fieldErrors.email"
            id="reg-email-error"
            class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
          >
            Error: {{ fieldErrors.email }}
          </p>
        </div>

        <div>
          <label for="reg-phone" class="mb-1.5 block text-sm font-semibold text-[var(--ink-primary)]">
            Nomor Telepon (Opsional)
          </label>
          <input
            id="reg-phone"
            v-model="formState.phone"
            name="phone"
            type="tel"
            autocomplete="tel"
            enterkeyhint="next"
            class="field-input"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label for="reg-password" class="mb-1 block text-sm font-semibold text-[var(--ink-primary)]">
            Kata Sandi
          </label>
          <span id="reg-password-hint" class="mb-1.5 block text-xs text-[var(--ink-secondary)]">
            Minimal 6 karakter.
          </span>
          <div class="relative">
            <input
              id="reg-password"
              v-model="formState.password"
              name="password"
              :type="showPassword ? 'text' : 'password'"
              autocomplete="new-password"
              minlength="6"
              enterkeyhint="next"
              required
              :aria-invalid="Boolean(fieldErrors.password)"
              :aria-describedby="fieldErrors.password ? 'reg-password-hint reg-password-error' : 'reg-password-hint'"
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
            id="reg-password-error"
            class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
          >
            Error: {{ fieldErrors.password }}
          </p>
        </div>

        <div>
          <label for="reg-pin" class="mb-1 block text-sm font-semibold text-[var(--ink-primary)]">
            PIN Kasir 6 Digit (Opsional)
          </label>
          <span id="reg-pin-hint" class="mb-1.5 block text-xs text-[var(--ink-secondary)]">
            Untuk login cepat kasir &amp; absensi kiosk.
          </span>
          <input
            id="reg-pin"
            v-model="formState.pin"
            name="pin"
            type="password"
            inputmode="numeric"
            pattern="[0-9]{6}"
            maxlength="6"
            autocomplete="off"
            enterkeyhint="done"
            :aria-invalid="Boolean(fieldErrors.pin)"
            :aria-describedby="fieldErrors.pin ? 'reg-pin-hint reg-pin-error' : 'reg-pin-hint'"
            class="field-input tracking-widest"
            @blur="onBlur"
            @input="onInput"
          />
          <p
            v-if="fieldErrors.pin"
            id="reg-pin-error"
            class="mt-1.5 text-sm font-medium text-[var(--error-text)]"
          >
            Error: {{ fieldErrors.pin }}
          </p>
        </div>
      </div>

      <fieldset class="rounded-md border border-[var(--border-ledger)] bg-[var(--bg-paper)] p-3.5">
        <legend class="px-1 text-xs font-bold uppercase tracking-wide text-[var(--ink-secondary)]">
          Identitas Usaha / Tenant (Opsional)
        </legend>
        <div class="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label for="reg-tenant-code" class="mb-1 block text-xs font-semibold text-[var(--ink-primary)]">
              Kode Tenant
            </label>
            <input
              id="reg-tenant-code"
              v-model="formState.tenant_code"
              name="tenant_code"
              type="text"
              autocorrect="off"
              autocapitalize="none"
              spellcheck="false"
              class="field-input"
            />
          </div>
          <div>
            <label for="reg-tenant-name" class="mb-1 block text-xs font-semibold text-[var(--ink-primary)]">
              Nama Usaha
            </label>
            <input
              id="reg-tenant-name"
              v-model="formState.tenant_name"
              name="tenant_name"
              type="text"
              class="field-input"
            />
          </div>
        </div>
      </fieldset>

      <button
        type="submit"
        :disabled="isSubmitting"
        class="mt-2 inline-flex min-h-[48px] w-full items-center justify-center rounded-md bg-[var(--brand-forest)] px-5 py-3 text-base font-semibold text-white transition-colors hover:bg-[var(--brand-forest-hover)] disabled:opacity-65"
      >
        {{ isSubmitting ? 'Mendaftarkan akun...' : 'Buat Akun' }}
      </button>
    </form>

    <div class="mt-6 border-t border-[var(--border-ledger)] pt-5 text-sm text-[var(--ink-secondary)]">
      Sudah terdaftar sebelumnya?
      <RouterLink to="/login" class="font-semibold text-[var(--accent-terracotta)] underline underline-offset-4">
        Masuk di sini
      </RouterLink>
    </div>
  </AuthLayout>
</template>
