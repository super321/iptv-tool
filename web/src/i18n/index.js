import { createI18n } from 'vue-i18n'
import axios from 'axios'

// Map browser language tags to supported locale codes
const LANG_MAP = {
  'zh-tw': 'zh-Hant',
  'zh-hk': 'zh-Hant',
  'zh-mo': 'zh-Hant',
  'zh-hant': 'zh-Hant',
  'zh-cn': 'zh',
  'zh-sg': 'zh',
  'zh-hans': 'zh',
}

// Detect the best language to use
function detectLanguage() {
  const saved = localStorage.getItem('locale')
  if (saved) return saved

  const browserLang = (navigator.language || navigator.languages?.[0] || 'en').toLowerCase()

  // Check full tag first (e.g. zh-tw, zh-cn)
  if (LANG_MAP[browserLang]) return LANG_MAP[browserLang]

  // Check prefix match for zh-Hant-XX or zh-Hans-XX variants
  for (const [prefix, locale] of Object.entries(LANG_MAP)) {
    if (browserLang.startsWith(prefix)) return locale
  }

  // For bare 'zh' (no region/script), default to Simplified Chinese
  const shortCode = browserLang.split('-')[0]
  if (shortCode === 'zh') return 'zh'

  // For non-Chinese languages, use the short code (en, fr, etc.)
  return shortCode
}

// Map locale codes to HTML lang attribute values
function getHtmlLang(lang) {
  if (lang === 'zh') return 'zh-CN'
  if (lang === 'zh-Hant') return 'zh-TW'
  return lang
}

const i18n = createI18n({
  legacy: false,
  locale: detectLanguage(),
  fallbackLocale: 'en',
  messages: {}
})

let loadedLocales = []

// Load locale messages from the backend API
export async function loadLocale(lang) {
  if (loadedLocales.includes(lang)) {
    i18n.global.locale.value = lang
    localStorage.setItem('locale', lang)
    document.documentElement.lang = getHtmlLang(lang)
    return
  }

  try {
    const { data } = await axios.get(`/api/locales/${lang}?t=${Date.now()}`)
    
    if (typeof data !== 'object' || data === null) {
      throw new Error('Received non-JSON response for locale data')
    }

    i18n.global.setLocaleMessage(lang, data)
    loadedLocales.push(lang)
    i18n.global.locale.value = lang
    localStorage.setItem('locale', lang)
    document.documentElement.lang = getHtmlLang(lang)
  } catch (e) {
    if (lang !== 'en') {
      console.warn(`Failed to load locale: ${lang}, falling back to en`)
      await loadLocale('en')
    } else {
      console.error(`Failed to load default locale en`, e)
    }
  }
}

// Get the current locale
export function getCurrentLocale() {
  return i18n.global.locale.value
}

export default i18n
