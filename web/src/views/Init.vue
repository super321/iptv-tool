<template>
  <div class="login-wrapper">
    <div class="login-bg">
      <div class="login-card">
        <div class="login-header">
          <div class="login-logo">
            <el-icon :size="36" color="#409eff"><Setting /></el-icon>
          </div>
          <h2 class="login-title">IPTV Tool</h2>
          <p class="login-subtitle">系统首次运行初始化</p>
        </div>
        <el-form :model="form" :rules="rules" ref="formRef" size="large">
          <el-form-item prop="username">
            <el-input v-model="form.username" placeholder="请设置管理员用户名 (至少3位)" :prefix-icon="User" />
          </el-form-item>
          <el-form-item prop="password">
            <el-input v-model="form.password" type="password" placeholder="请设置登录密码 (至少6位)" :prefix-icon="Lock" show-password />
          </el-form-item>
          <el-form-item prop="confirmPassword">
            <el-input v-model="form.confirmPassword" type="password" placeholder="请再次确认密码" :prefix-icon="Lock" show-password @keyup.enter="handleInit" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleInit" :loading="loading" style="width: 100%" size="large">初始化并保存</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'
import { User, Lock, Setting } from '@element-plus/icons-vue'

const router = useRouter()
const auth = useAuthStore()
const formRef = ref()
const loading = ref(false)

const form = reactive({ username: '', password: '', confirmPassword: '' })

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }, { min: 3, message: '至少3个字符', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }, { min: 6, message: '至少6个字符', trigger: 'blur' }],
  confirmPassword: [{
    required: true, message: '请确认密码', trigger: 'blur',
  }, {
    validator: (_, val, cb) => val === form.password ? cb() : cb(new Error('两次密码不一致')),
    trigger: 'blur',
  }],
}

async function handleInit() {
  await formRef.value.validate()
  loading.value = true
  try {
    await auth.init(form.username, form.password)
    ElMessage.success('初始化成功，请登录')
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
</style>