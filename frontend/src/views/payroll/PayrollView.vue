<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ApiError, apiRequest } from '../../api/client'
import AppShell from '../../components/AppShell.vue'
import { useAuthStore } from '../../stores/auth'
import { formatDate, formatDateTime, formatIDR } from '../../utils/format'

interface EmployeeItem {
  id: number
  uuid: string
  employee_code: string
  full_name: string
  job_title: string
  employment_status: string
  is_active: boolean
  salary_structure?: {
    basic_salary: number
    fixed_allowance: number
    meal_transport_allowance: number
    commission_rate_percent: number
    commission_fixed_per_item: number
    bpjs_health_deduction: number
    bpjs_employment_deduction: number
  }
}

interface AttendanceLog {
  uuid: string
  log_date: string
  clock_in?: string
  clock_out?: string
  late_minutes: number
  overtime_minutes: number
  status: string
  employee?: { full_name: string; employee_code: string }
}

interface CommissionItem {
  uuid: string
  order_id: number
  commission_amount: number
  status: string
  created_at: string
  employee?: { full_name: string; employee_code: string }
}

interface PayrollPeriod {
  uuid: string
  period_name: string
  start_date: string
  end_date: string
  payment_date: string
  status: string
  total_payout: number
}

interface PayslipSnapshot {
  uuid: string
  employee_code: string
  employee_name: string
  job_title: string
  basic_salary: number
  total_allowances: number
  total_overtime_pay: number
  total_commissions: number
  total_deductions: number
  net_salary: number
  status: string
}

const auth = useAuthStore()
const canFinalize = computed(() => auth.user?.role === 'SUPERADMIN' || auth.permissions.includes('payroll:finalize'))

const activeTab = ref<'employees' | 'attendance' | 'commissions' | 'periods'>('employees')
const loading = ref(false)
const errorMsg = ref('')
const actionMsg = ref('')

const employees = ref<EmployeeItem[]>([])
const attendanceLogs = ref<AttendanceLog[]>([])
const commissions = ref<CommissionItem[]>([])
const periods = ref<PayrollPeriod[]>([])
const slips = ref<PayslipSnapshot[]>([])

const empForm = reactive({
  outlet_id: 1,
  employee_code: '',
  full_name: '',
  job_title: '',
  joined_date: new Date().toISOString().slice(0, 10),
  basic_salary: 3000000,
  commission_rate_percent: 2,
})

const kioskForm = reactive({
  employee_uuid: '',
  outlet_id: 1,
  pin: '',
  late_minutes: 0,
  overtime_minutes: 0,
})

const periodForm = reactive({
  period_name: '',
  start_date: new Date().toISOString().slice(0, 8) + '01',
  end_date: new Date().toISOString().slice(0, 10),
  payment_date: new Date().toISOString().slice(0, 10),
})

async function loadAll() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [empRes, attRes, commRes, perRes, slipRes] = await Promise.all([
      apiRequest<{ items: EmployeeItem[] }>('/api/v1/payroll/employees?limit=50', { method: 'GET' }, auth.accessToken),
      apiRequest<{ items: AttendanceLog[] }>('/api/v1/payroll/attendance?limit=30', { method: 'GET' }, auth.accessToken),
      apiRequest<{ items: CommissionItem[] }>('/api/v1/payroll/commissions?limit=30', { method: 'GET' }, auth.accessToken),
      apiRequest<{ items: PayrollPeriod[] }>('/api/v1/payroll/periods?limit=20', { method: 'GET' }, auth.accessToken),
      apiRequest<{ items: PayslipSnapshot[] }>('/api/v1/payroll/slips?limit=30', { method: 'GET' }, auth.accessToken),
    ])
    employees.value = empRes.items || []
    attendanceLogs.value = attRes.items || []
    commissions.value = commRes.items || []
    periods.value = perRes.items || []
    slips.value = slipRes.items || []
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal memuat data absensi dan penggajian.'
  } finally {
    loading.value = false
  }
}

async function handleCreateEmployee() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/payroll/employees',
      {
        method: 'POST',
        body: JSON.stringify({
          outlet_id: auth.outlets[0]?.id || 1,
          employee_code: empForm.employee_code.trim(),
          full_name: empForm.full_name.trim(),
          job_title: empForm.job_title.trim(),
          joined_date: empForm.joined_date,
          salary_structure: {
            basic_salary: Number(empForm.basic_salary),
            commission_rate_percent: Number(empForm.commission_rate_percent),
          },
        }),
      },
      auth.accessToken,
    )
    actionMsg.value = `Karyawan "${empForm.full_name}" berhasil didaftarkan.`
    empForm.employee_code = ''
    empForm.full_name = ''
    empForm.job_title = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal mendaftarkan karyawan.'
  }
}

async function handleClockIn() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/payroll/attendance/clock-in',
      {
        method: 'POST',
        body: JSON.stringify({
          employee_uuid: kioskForm.employee_uuid,
          outlet_id: auth.outlets[0]?.id || 1,
          pin: kioskForm.pin.trim() || undefined,
          late_minutes: Number(kioskForm.late_minutes),
        }),
      },
      auth.accessToken,
    )
    actionMsg.value = 'Clock-in absensi berhasil dicatat.'
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Clock-in gagal.'
  }
}

async function handleClockOut() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      '/api/v1/payroll/attendance/clock-out',
      {
        method: 'POST',
        body: JSON.stringify({
          employee_uuid: kioskForm.employee_uuid,
          pin: kioskForm.pin.trim() || undefined,
          overtime_minutes: Number(kioskForm.overtime_minutes),
        }),
      },
      auth.accessToken,
    )
    actionMsg.value = 'Clock-out absensi berhasil dicatat.'
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Clock-out gagal.'
  }
}

async function handleCreatePeriod() {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest('/api/v1/payroll/periods', { method: 'POST', body: JSON.stringify(periodForm) }, auth.accessToken)
    actionMsg.value = `Periode "${periodForm.period_name}" berhasil dibuat.`
    periodForm.period_name = ''
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal membuat periode penggajian.'
  }
}

async function handleFinalizePeriod(p: PayrollPeriod) {
  errorMsg.value = ''
  actionMsg.value = ''
  try {
    await apiRequest(
      `/api/v1/payroll/periods/${p.uuid}/finalize`,
      { method: 'POST', body: JSON.stringify({ overtime_rate_per_hour: 25000, late_penalty_per_minute: 1000 }) },
      auth.accessToken,
    )
    actionMsg.value = `Periode "${p.period_name}" dikunci menjadi FINALIZED dan slip gaji dibuat.`
    await loadAll()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : 'Gagal memfinalisasi periode.'
  }
}

async function handleDownloadPDF(slip: PayslipSnapshot) {
  errorMsg.value = ''
  try {
    const res = await fetch(`/api/v1/payroll/slips/${slip.uuid}/pdf`, {
      headers: { Authorization: `Bearer ${auth.accessToken}` },
    })
    if (!res.ok) throw new Error('Gagal mengunduh PDF slip gaji.')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `slip-gaji-${slip.employee_code}.pdf`
    link.click()
    URL.revokeObjectURL(url)
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : 'Gagal mengunduh PDF.'
  }
}

onMounted(loadAll)
</script>

<template>
  <AppShell page-title="Absensi &amp; Penggajian" page-subtitle="Data karyawan, absensi kiosk, komisi POS, finalisasi gaji immutable, dan slip PDF.">
    <div v-if="errorMsg" role="alert" class="mb-4 rounded-md border border-[var(--error-text)] bg-[var(--error-bg)] px-4 py-3 text-sm font-medium text-[var(--error-text)]">
      Error: {{ errorMsg }}
    </div>
    <div v-if="actionMsg" role="status" class="mb-4 rounded-md border border-[var(--brand-forest)] bg-[var(--bg-surface)] px-4 py-3 text-sm font-semibold text-[var(--brand-forest)]">
      {{ actionMsg }}
    </div>

    <div role="tablist" aria-label="Menu payroll" class="mb-5 flex flex-wrap gap-2 rounded-lg border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-1 w-fit">
      <button
        v-for="tab in [
          { id: 'employees', label: `Karyawan (${employees.length})` },
          { id: 'attendance', label: `Absensi (${attendanceLogs.length})` },
          { id: 'commissions', label: `Komisi (${commissions.length})` },
          { id: 'periods', label: `Periode & Slip (${periods.length})` },
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

    <div v-if="loading" role="status" class="py-6 text-sm text-[var(--ink-secondary)]">Memuat data payroll...</div>

    <!-- Employees -->
    <div v-else-if="activeTab === 'employees'" class="space-y-6">
      <section v-if="canFinalize" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Registrasi Karyawan &amp; Gaji Pokok</h2>
        <form class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-6" @submit.prevent="handleCreateEmployee">
          <div>
            <label for="emp-code" class="mb-1 block text-xs font-semibold">Kode</label>
            <input id="emp-code" v-model="empForm.employee_code" required placeholder="EMP-01" class="field-input" />
          </div>
          <div>
            <label for="emp-name" class="mb-1 block text-xs font-semibold">Nama Lengkap</label>
            <input id="emp-name" v-model="empForm.full_name" required class="field-input" />
          </div>
          <div>
            <label for="emp-job" class="mb-1 block text-xs font-semibold">Jabatan</label>
            <input id="emp-job" v-model="empForm.job_title" required placeholder="Kasir" class="field-input" />
          </div>
          <div>
            <label for="emp-sal" class="mb-1 block text-xs font-semibold">Gaji Pokok</label>
            <input id="emp-sal" v-model.number="empForm.basic_salary" type="number" min="0" required class="field-input" />
          </div>
          <div>
            <label for="emp-comm" class="mb-1 block text-xs font-semibold">Komisi (%)</label>
            <input id="emp-comm" v-model.number="empForm.commission_rate_percent" type="number" step="0.1" min="0" max="100" class="field-input" />
          </div>
          <div class="flex items-end">
            <button type="submit" class="min-h-[48px] w-full rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
              Simpan
            </button>
          </div>
        </form>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Daftar Karyawan</h2>
        <p v-if="!employees.length" class="text-sm text-[var(--ink-secondary)]">Belum ada data karyawan.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
                <th scope="col" class="px-3 py-2.5">Kode</th>
                <th scope="col" class="px-3 py-2.5">Nama</th>
                <th scope="col" class="px-3 py-2.5">Jabatan</th>
                <th scope="col" class="px-3 py-2.5">Gaji Pokok</th>
                <th scope="col" class="px-3 py-2.5">Komisi</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="e in employees" :key="e.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
                <th scope="row" class="px-3 py-2.5 font-semibold">{{ e.employee_code }}</th>
                <td class="px-3 py-2.5">{{ e.full_name }}</td>
                <td class="px-3 py-2.5 text-[var(--ink-secondary)]">{{ e.job_title }}</td>
                <td class="px-3 py-2.5 font-semibold">{{ formatIDR(e.salary_structure?.basic_salary) }}</td>
                <td class="px-3 py-2.5">{{ e.salary_structure?.commission_rate_percent || 0 }}%</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <!-- Attendance -->
    <div v-else-if="activeTab === 'attendance'" class="space-y-6">
      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Kiosk Absensi (Clock-In / Clock-Out)</h2>
        <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-4">
          <div>
            <label for="att-emp" class="mb-1 block text-xs font-semibold">Karyawan</label>
            <select id="att-emp" v-model="kioskForm.employee_uuid" class="field-input">
              <option value="" disabled>Pilih Karyawan</option>
              <option v-for="e in employees" :key="e.uuid" :value="e.uuid">{{ e.full_name }} ({{ e.employee_code }})</option>
            </select>
          </div>
          <div>
            <label for="att-pin" class="mb-1 block text-xs font-semibold">PIN (jika tertaut User)</label>
            <input id="att-pin" v-model="kioskForm.pin" type="password" inputmode="numeric" maxlength="6" class="field-input" />
          </div>
          <div>
            <label for="att-late" class="mb-1 block text-xs font-semibold">Terlambat (Menit)</label>
            <input id="att-late" v-model.number="kioskForm.late_minutes" type="number" min="0" class="field-input" />
          </div>
          <div>
            <label for="att-ot" class="mb-1 block text-xs font-semibold">Lembur (Menit)</label>
            <input id="att-ot" v-model.number="kioskForm.overtime_minutes" type="number" min="0" class="field-input" />
          </div>
        </div>
        <div class="mt-3 flex flex-wrap gap-2">
          <button type="button" :disabled="!kioskForm.employee_uuid" class="min-h-[44px] rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white disabled:opacity-50" @click="handleClockIn">
            Clock-In
          </button>
          <button type="button" :disabled="!kioskForm.employee_uuid" class="min-h-[44px] rounded-md bg-[var(--accent-terracotta)] px-4 py-2 text-sm font-semibold text-white disabled:opacity-50" @click="handleClockOut">
            Clock-Out
          </button>
        </div>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Log Absensi</h2>
        <p v-if="!attendanceLogs.length" class="text-sm text-[var(--ink-secondary)]">Belum ada log absensi.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
                <th scope="col" class="px-3 py-2.5">Tanggal</th>
                <th scope="col" class="px-3 py-2.5">Karyawan</th>
                <th scope="col" class="px-3 py-2.5">Masuk</th>
                <th scope="col" class="px-3 py-2.5">Pulang</th>
                <th scope="col" class="px-3 py-2.5">Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in attendanceLogs" :key="a.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
                <th scope="row" class="px-3 py-2.5 font-semibold">{{ formatDate(a.log_date) }}</th>
                <td class="px-3 py-2.5">{{ a.employee?.full_name || '-' }}</td>
                <td class="px-3 py-2.5">{{ a.clock_in ? formatDateTime(a.clock_in) : '-' }}</td>
                <td class="px-3 py-2.5">{{ a.clock_out ? formatDateTime(a.clock_out) : '-' }}</td>
                <td class="px-3 py-2.5 font-semibold">{{ a.status }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <!-- Commissions -->
    <section v-else-if="activeTab === 'commissions'" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
      <h2 class="mb-3 text-base font-bold">Antrean Komisi Penjualan POS</h2>
      <p v-if="!commissions.length" class="text-sm text-[var(--ink-secondary)]">Belum ada komisi penjualan tercatat.</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
              <th scope="col" class="px-3 py-2.5">Karyawan</th>
              <th scope="col" class="px-3 py-2.5">Order ID</th>
              <th scope="col" class="px-3 py-2.5">Komisi</th>
              <th scope="col" class="px-3 py-2.5">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in commissions" :key="c.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
              <th scope="row" class="px-3 py-2.5 font-semibold">{{ c.employee?.full_name || '-' }}</th>
              <td class="px-3 py-2.5">#{{ c.order_id }}</td>
              <td class="px-3 py-2.5 font-semibold text-[var(--brand-forest)]">{{ formatIDR(c.commission_amount) }}</td>
              <td class="px-3 py-2.5">{{ c.status }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Periods & Payslips -->
    <div v-else class="space-y-6">
      <section v-if="canFinalize" class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="text-base font-bold">Buka Periode Penggajian Baru</h2>
        <form class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-5" @submit.prevent="handleCreatePeriod">
          <div>
            <label for="per-name" class="mb-1 block text-xs font-semibold">Nama Periode</label>
            <input id="per-name" v-model="periodForm.period_name" required placeholder="November 2026" class="field-input" />
          </div>
          <div>
            <label for="per-start" class="mb-1 block text-xs font-semibold">Mulai</label>
            <input id="per-start" v-model="periodForm.start_date" type="date" required class="field-input" />
          </div>
          <div>
            <label for="per-end" class="mb-1 block text-xs font-semibold">Selesai</label>
            <input id="per-end" v-model="periodForm.end_date" type="date" required class="field-input" />
          </div>
          <div>
            <label for="per-pay" class="mb-1 block text-xs font-semibold">Tgl Bayar</label>
            <input id="per-pay" v-model="periodForm.payment_date" type="date" required class="field-input" />
          </div>
          <div class="flex items-end">
            <button type="submit" class="min-h-[48px] w-full rounded-md bg-[var(--brand-forest)] px-4 py-2 text-sm font-semibold text-white">
              Buat Periode
            </button>
          </div>
        </form>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Daftar Periode Penggajian</h2>
        <p v-if="!periods.length" class="text-sm text-[var(--ink-secondary)]">Belum ada periode penggajian.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
                <th scope="col" class="px-3 py-2.5">Periode</th>
                <th scope="col" class="px-3 py-2.5">Rentang</th>
                <th scope="col" class="px-3 py-2.5">Total Payout</th>
                <th scope="col" class="px-3 py-2.5">Status</th>
                <th scope="col" class="px-3 py-2.5">Aksi</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in periods" :key="p.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
                <th scope="row" class="px-3 py-2.5 font-semibold">{{ p.period_name }}</th>
                <td class="px-3 py-2.5 text-xs text-[var(--ink-secondary)]">{{ formatDate(p.start_date) }} s/d {{ formatDate(p.end_date) }}</td>
                <td class="px-3 py-2.5 font-semibold">{{ formatIDR(p.total_payout) }}</td>
                <td class="px-3 py-2.5 font-semibold">{{ p.status }}</td>
                <td class="px-3 py-2.5">
                  <button
                    v-if="canFinalize && p.status !== 'FINALIZED'"
                    type="button"
                    class="min-h-[36px] rounded bg-[var(--brand-forest)] px-3 py-1 text-xs font-semibold text-white"
                    @click="handleFinalizePeriod(p)"
                  >
                    Finalisasi &amp; Kunci
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="rounded-[10px] border border-[var(--border-ledger)] bg-[var(--bg-surface)] p-5">
        <h2 class="mb-3 text-base font-bold">Slip Gaji Final (Immutable Snapshots)</h2>
        <p v-if="!slips.length" class="text-sm text-[var(--ink-secondary)]">Belum ada slip gaji yang difinalisasi.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-[var(--border-ledger)] bg-[var(--bg-paper)] text-xs font-bold uppercase text-[var(--ink-secondary)]">
                <th scope="col" class="px-3 py-2.5">Kode</th>
                <th scope="col" class="px-3 py-2.5">Nama</th>
                <th scope="col" class="px-3 py-2.5">Gaji Bersih</th>
                <th scope="col" class="px-3 py-2.5">Unduh</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in slips" :key="s.uuid" class="border-b border-[var(--border-ledger)] last:border-b-0">
                <th scope="row" class="px-3 py-2.5 font-semibold">{{ s.employee_code }}</th>
                <td class="px-3 py-2.5">{{ s.employee_name }} ({{ s.job_title }})</td>
                <td class="px-3 py-2.5 font-bold text-[var(--brand-forest)]">{{ formatIDR(s.net_salary) }}</td>
                <td class="px-3 py-2.5">
                  <button type="button" class="min-h-[36px] rounded border border-[var(--border-strong)] px-3 py-1 text-xs font-semibold" @click="handleDownloadPDF(s)">
                    Unduh PDF
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
