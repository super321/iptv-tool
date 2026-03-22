<template>
  <div class="login-wrapper">
    <div class="login-bg">
      <div class="lang-switch">
        <el-dropdown @command="switchLanguage">
          <span class="lang-dropdown">
            {{ $t('language.' + currentLocale) }}
            <el-icon style="margin-left: 4px"><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="zh">{{ $t('language.zh') }}</el-dropdown-item>
              <el-dropdown-item command="zh-Hant">{{ $t('language.zh-Hant') }}</el-dropdown-item>
              <el-dropdown-item command="en">{{ $t('language.en') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="login-card">
        <div class="login-header">
          <div class="login-logo">
            <el-icon :size="36" color="#409eff"><Monitor /></el-icon>
          </div>
          <h2 class="login-title">IPTV Tool</h2>
          <p class="login-subtitle">{{ $t('login.admin_login') }}</p>
        </div>
        <el-form :model="form" :rules="rules" ref="formRef" size="large">
          <el-form-item prop="username">
            <el-input v-model.trim="form.username" :placeholder="$t('login.username_placeholder')" :prefix-icon="User" @keyup.enter="handleLogin" />
          </el-form-item>
          <el-form-item prop="password">
            <el-input v-model="form.password" type="password" :placeholder="$t('login.password_placeholder')" :prefix-icon="Lock" show-password @keyup.enter="handleLogin" />
          </el-form-item>
          <!-- 验证码区域 - 动态显示 -->
          <el-form-item v-if="captchaRequired" prop="captchaCode">
            <div class="captcha-row">
              <el-input v-model="form.captchaCode" :placeholder="$t('login.captcha_placeholder')" :prefix-icon="Key" @keyup.enter="handleLogin" />
              <img
                v-if="captchaImage"
                :src="captchaImage"
                class="captcha-img"
                @click="refreshCaptcha"
                :title="$t('login.captcha_refresh')"
                :alt="$t('login.captcha_alt')"
              />
              <div v-else class="captcha-placeholder" @click="refreshCaptcha">{{ $t('common.loading') }}</div>
            </div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleLogin" :loading="loading" style="width: 100%" size="large">{{ $t('login.login_btn') }}</el-button>
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
import { ElMessage } from 'element-plus'
import { User, Lock, Key, Monitor, ArrowDown } from '@element-plus/icons-vue'

const router = useRouter()
const auth = useAuthStore()
const { t, locale } = useI18n()
const currentLocale = computed(() => locale.value)
const formRef = ref()
const loading = ref(false)

async function switchLanguage(lang) {
  await loadLocale(lang)
}

// 验证码状态
const captchaRequired = ref(false)
const captchaId = ref('')
const captchaImage = ref('')

const form = reactive({ username: '', password: '', captchaCode: '' })
const rules = computed(() => ({
  username: [{ required: true, message: t('login.required_username'), trigger: 'blur' }],
  password: [{ required: true, message: t('login.required_password'), trigger: 'blur' }],
  captchaCode: [{ required: true, message: t('login.required_captcha'), trigger: 'blur' }],
}))

// 获取/刷新验证码
async function refreshCaptcha() {
  try {
    const data = await auth.getCaptcha()
    captchaId.value = data.captcha_id
    captchaImage.value = data.captcha_image
    form.captchaCode = ''
  } catch {
    ElMessage.error(t('login.captcha_fetch_failed'))
  }
}

async function handleLogin() {
  await formRef.value.validate()
  loading.value = true
  try {
    await auth.login(
      form.username,
      form.password,
      captchaRequired.value ? captchaId.value : undefined,
      captchaRequired.value ? form.captchaCode : undefined
    )
    ElMessage.success(t('login.login_success'))
    router.push('/')
  } catch (err) {
    const resp = err.response
    if (!resp) return

    const { status, data } = resp
    if (status === 429) {
      ElMessage.error((data && data.error) || t('login.login_rate_limited'))
      return
    }

    // 401 (密码错误) 或 403 (验证码相关)
    if (data && data.error) {
      ElMessage.error(data.error)
    }

    // 后端指示需要验证码
    if (data && data.captcha_required) {
      captchaRequired.value = true
      await refreshCaptcha()
    } else if (captchaRequired.value) {
      // 已经在验证码模式下，刷新验证码供下次使用
      await refreshCaptcha()
    }
  } finally {
    loading.value = false
  }
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
  background: linear-gradient(135deg, #1d2b3a 0%, #2c3e50 40%, #34495e 70%, #1a252f 100%);
  position: relative;
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
}
.lang-dropdown {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: rgba(255, 255, 255, 0.8);
  font-size: 14px;
  font-weight: 500;
  transition: color 0.3s;
}
.lang-dropdown:hover {
  color: #fff;
}
.login-card {
  width: 400px;
  padding: 40px 36px 28px;
  background: rgba(255, 255, 255, 0.95);
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
  color: #1d2b3a;
  margin: 0 0 6px;
  letter-spacing: 1px;
}
.login-subtitle {
  font-size: 14px;
  color: #909399;
  margin: 0;
}
.captcha-row {
  display: flex;
  gap: 8px;
  width: 100%;
  align-items: center;
}
.captcha-row .el-input {
  flex: 1;
}
.captcha-img {
  height: 40px;
  border-radius: 4px;
  cursor: pointer;
  flex-shrink: 0;
}
.captcha-placeholder {
  height: 40px;
  width: 120px;
  border-radius: 4px;
  background: #f5f5f5;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: #999;
  cursor: pointer;
  flex-shrink: 0;
}
</style>
