<template>
  <div class="login-wrapper">
    <div class="login-bg">
      <div class="lang-switch">
        <el-dropdown @command="switchLanguage">
          <span class="lang-dropdown">
            <svg class="lang-icon-login" viewBox="0 0 24 24" width="18" height="18">
              <text x="2" y="15" font-size="14" font-weight="600" font-family="sans-serif" fill="currentColor">文</text>
              <text x="14" y="20" font-size="10" font-weight="700" font-family="sans-serif" fill="currentColor">A</text>
            </svg>
            <el-icon style="margin-left: 2px" :size="12"><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="zh">{{ $t('language.zh') }}</el-dropdown-item>
              <el-dropdown-item command="zh-Hant">{{ $t('language.zh-Hant') }}</el-dropdown-item>
              <el-dropdown-item command="en">{{ $t('language.en') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <div class="theme-toggle-login">
          <el-switch
            v-model="isDarkTheme"
            inline-prompt
            :active-icon="Moon"
            :inactive-icon="Sunny"
            class="login-theme-switch"
          />
        </div>
      </div>
      <div class="login-card">
        <div class="login-header">
          <div class="login-logo">
            <el-icon :size="36" color="#409eff"><Setting /></el-icon>
          </div>
          <h2 class="login-title">IPTV Tool</h2>
          <p class="login-subtitle">{{ $t('init.system_init') }}</p>
        </div>
        <el-form :model="form" :rules="rules" ref="formRef" size="large">
          <el-form-item prop="username">
            <el-input v-model.trim="form.username" :placeholder="$t('init.username_placeholder')" :prefix-icon="User" />
          </el-form-item>
          <el-form-item prop="password">
            <el-input v-model="form.password" type="password" :placeholder="$t('init.password_placeholder')" :prefix-icon="Lock" show-password />
          </el-form-item>
          <el-form-item prop="confirmPassword">
            <el-input v-model="form.confirmPassword" type="password" :placeholder="$t('init.confirm_password_placeholder')" :prefix-icon="Lock" show-password @keyup.enter="handleInit" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleInit" :loading="loading" style="width: 100%" size="large">{{ $t('init.init_btn') }}</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { loadLocale } from '../i18n'
import { useAuthStore } from '../stores/auth'
import { useThemeStore } from '../stores/theme'
import { ElMessage } from 'element-plus'
import { User, Lock, Setting, ArrowDown, Sunny, Moon } from '@element-plus/icons-vue'

const router = useRouter()
const auth = useAuthStore()
const themeStore = useThemeStore()
const { t, locale } = useI18n()
const currentLocale = computed(() => locale.value)
const formRef = ref()
const loading = ref(false)

const isDarkTheme = computed({
  get: () => themeStore.isDark,
  set: (val) => themeStore.setMode(val ? 'dark' : 'light')
})

async function switchLanguage(lang) {
  await loadLocale(lang)
}

const form = reactive({ username: '', password: '', confirmPassword: '' })

const rules = computed(() => ({
  username: [
    { required: true, message: t('init.required_username'), trigger: 'blur' },
    { min: 3, message: t('init.min_username'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('init.required_password'), trigger: 'blur' },
    { min: 6, message: t('init.min_password'), trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: t('init.required_confirm'), trigger: 'blur' },
    { validator: (_, val, cb) => val === form.password ? cb() : cb(new Error(t('init.password_mismatch'))), trigger: 'blur' }
  ],
}))

async function handleInit() {
  await formRef.value.validate()
  loading.value = true
  try {
    await auth.init(form.username, form.password)
    ElMessage.success(t('init.init_success'))
    router.push('/login')
  } catch { /* handled by interceptor */ }
  finally { loading.value = false }
}
</script>

<style scoped>
.login-wrapper {
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}
.login-bg {
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  background: var(--login-bg-gradient);
  position: relative;
  transition: background 0.3s;
}
.login-bg::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  background:
    radial-gradient(ellipse at 20% 50%, rgba(64, 158, 255, 0.08) 0%, transparent 50%),
    radial-gradient(ellipse at 80% 20%, rgba(64, 158, 255, 0.05) 0%, transparent 40%);
}
.lang-switch {
  position: absolute;
  top: 24px;
  right: 24px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 12px;
}
.lang-dropdown {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: rgba(255, 255, 255, 0.8);
  font-size: 14px;
  font-weight: 500;
  transition: color 0.3s;
  outline: none;
}
.lang-dropdown:focus {
  outline: none;
}
.lang-dropdown:hover {
  color: #fff;
}
.lang-icon-login {
  color: rgba(255, 255, 255, 0.8);
  transition: color 0.3s;
}
.lang-dropdown:hover .lang-icon-login {
  color: #fff;
}
.theme-toggle-login {
  cursor: pointer;
  display: flex;
  align-items: center;
}
.login-theme-switch {
  --el-switch-on-color: rgba(255,255,255,0.15);
  --el-switch-off-color: rgba(255,255,255,0.15);
  --el-switch-border-color: rgba(255,255,255,0.3);
}
:deep(.login-theme-switch .el-switch__core) {
  border: 1px solid rgba(255,255,255,0.3);
  background-color: rgba(255,255,255,0.15) !important;
  transition: box-shadow 0.3s;
}
:deep(.login-theme-switch .el-switch__action) {
  background-color: rgba(255,255,255,0.9);
  color: var(--el-text-color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
}
:deep(.login-theme-switch .el-switch__inner .el-icon) {
  color: rgba(255,255,255,0.5);
}
.theme-toggle-login:hover :deep(.login-theme-switch .el-switch__core) {
  box-shadow: 0 0 8px rgba(255, 255, 255, 0.3);
}
.login-card {
  width: 400px;
  padding: 40px 36px 28px;
  background: var(--login-card-bg);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  position: relative;
  z-index: 1;
}
.login-header {
  text-align: center;
  margin-bottom: 32px;
}
.login-logo {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  background: linear-gradient(135deg, #409eff 0%, #337ecc 100%);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.login-logo .el-icon {
  color: #fff !important;
}
.login-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--login-title-color);
  margin: 0 0 6px;
  letter-spacing: 1px;
}
.login-subtitle {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin: 0;
}
</style>