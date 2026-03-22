import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'

const STORAGE_KEY = 'iptv_theme'

export const useThemeStore = defineStore('theme', () => {
  // No stored value → follow browser preference; 'light'/'dark' → user override
  const mode = ref(localStorage.getItem(STORAGE_KEY) || 'system')

  const systemDark = ref(
    window.matchMedia('(prefers-color-scheme: dark)').matches
  )

  // Listen for system theme changes
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', (e) => {
    systemDark.value = e.matches
  })

  const isDark = computed(() => {
    if (mode.value === 'dark') return true
    if (mode.value === 'light') return false
    return systemDark.value
  })

  function applyTheme() {
    if (isDark.value) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  function setMode(newMode) {
    mode.value = newMode
    localStorage.setItem(STORAGE_KEY, newMode)
  }

  // Watch and apply whenever isDark changes
  watch(isDark, () => applyTheme(), { immediate: true })

  // 'system' is only the default when user has never set a preference.
  // Once user explicitly switches, setMode('light'/'dark') is used.
  return { mode, isDark, setMode }
})
