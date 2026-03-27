import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
  },
  {
    path: '/init',
    name: 'Init',
    component: () => import('../views/Init.vue'),
  },
  {
    path: '/',
    component: () => import('../layout/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/live-sources' },
      {
        path: 'live-sources',
        name: 'LiveSources',
        component: () => import('../views/LiveSources.vue'),
      },
      {
        path: 'epg-sources',
        name: 'EPGSources',
        component: () => import('../views/EPGSources.vue'),
      },
      {
        path: 'logos',
        name: 'Logos',
        component: () => import('../views/Logos.vue'),
      },
      {
        path: 'rules',
        name: 'Rules',
        component: () => import('../views/Rules.vue'),
      },
      {
        path: 'publish',
        name: 'Publish',
        component: () => import('../views/Publish.vue'),
      },
      {
        path: 'logs/runtime',
        name: 'LogRuntime',
        component: () => import('../views/LogRuntime.vue'),
      },
      {
        path: 'logs/access',
        name: 'LogAccess',
        component: () => import('../views/LogAccess.vue'),
      },
      {
        path: 'settings/detect',
        name: 'SettingsDetect',
        component: () => import('../views/SettingsDetect.vue'),
      },
      {
        path: 'settings/access-control',
        name: 'SettingsAccessControl',
        component: () => import('../views/SettingsAccessControl.vue'),
      },
      {
        path: 'settings/password',
        name: 'SettingsPassword',
        component: () => import('../views/SettingsPassword.vue'),
      },
      {
        path: 'settings/data',
        name: 'ConfigTransfer',
        component: () => import('../views/ConfigTransfer.vue'),
      },
      {
        path: 'settings/about',
        name: 'SettingsAbout',
        component: () => import('../views/SettingsAbout.vue'),
      },
    ],
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  // Check system initialization on first load
  if (auth.initialized === null) {
    try {
      await auth.checkInit()
    } catch {
      // network error, proceed
    }
  }

  // Redirect to init page if system not initialized
  if (auth.initialized === false && to.name !== 'Init') {
    return { name: 'Init' }
  }

  // Prevent accessing Init page if system is already initialized
  if (auth.initialized === true && to.name === 'Init') {
    return { name: 'Login' }
  }

  // Redirect to login if auth required and not logged in
  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return { name: 'Login' }
  }
})

export default router
