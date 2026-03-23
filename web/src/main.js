import { createApp } from 'vue'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import i18n, { loadLocale } from './i18n'
import './style.css'

// Global tooltip for truncated table headers
document.addEventListener('mouseover', (e) => {
  if (!e.target || !e.target.closest) return;
  const cell = e.target.closest('.el-table th.el-table__cell > .cell');
  if (cell) {
    if (cell.scrollWidth > cell.clientWidth) {
      if (!cell.hasAttribute('title')) {
        cell.setAttribute('title', cell.innerText);
      }
    } else {
      cell.removeAttribute('title');
    }
  }
});

const app = createApp(App)

app.use(createPinia())
app.use(i18n)
app.use(router)

// Load initial locale before mounting
loadLocale(i18n.global.locale.value).then(() => {
  app.mount('#app')
})
