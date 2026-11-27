import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/auth/LoginView.vue'),
      meta: { title: 'Masuk | UMKM POS', guestOnly: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('../views/auth/RegisterView.vue'),
      meta: { title: 'Daftar Akun | UMKM POS', guestOnly: true },
    },
    {
      path: '/',
      name: 'workspace',
      component: () => import('../views/DashboardView.vue'),
      meta: { title: 'Dashboard Operasional | UMKM POS', requiresAuth: true },
    },
    {
      path: '/pos',
      name: 'pos',
      component: () => import('../views/pos/PosView.vue'),
      meta: { title: 'Kasir POS & Shift | UMKM POS', requiresAuth: true },
    },
    {
      path: '/products',
      name: 'products',
      component: () => import('../views/products/ProductsView.vue'),
      meta: { title: 'Katalog Produk | UMKM POS', requiresAuth: true },
    },
    {
      path: '/inventory',
      name: 'inventory',
      component: () => import('../views/inventory/InventoryView.vue'),
      meta: { title: 'Stok & Pembelian | UMKM POS', requiresAuth: true },
    },
    {
      path: '/promotions',
      name: 'promotions',
      component: () => import('../views/promotions/PromotionsView.vue'),
      meta: { title: 'Promosi & Pelanggan | UMKM POS', requiresAuth: true },
    },
    {
      path: '/payroll',
      name: 'payroll',
      component: () => import('../views/payroll/PayrollView.vue'),
      meta: { title: 'Absensi & Penggajian | UMKM POS', requiresAuth: true },
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && auth.isAuthenticated) {
    return { name: 'workspace' }
  }
  return true
})

router.afterEach((to) => {
  const title = to.meta.title as string | undefined
  if (title) {
    document.title = title
  }
})

export default router
