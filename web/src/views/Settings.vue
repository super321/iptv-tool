<template>
  <div>
    <h3 style="margin: 0 0 20px">系统设置</h3>

    <el-row :gutter="20">
      <!-- Password Card -->
      <el-col :span="12" :xs="24" style="margin-bottom: 20px">
        <el-card shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <el-icon :size="18"><Lock /></el-icon>
              <span>修改密码</span>
            </div>
          </template>
          <el-form :model="pwdForm" :rules="pwdRules" ref="pwdFormRef" label-width="100px">
            <el-form-item label="当前密码" prop="oldPassword">
              <el-input v-model="pwdForm.oldPassword" type="password" show-password />
            </el-form-item>
            <el-form-item label="新密码" prop="newPassword">
              <el-input v-model="pwdForm.newPassword" type="password" show-password />
            </el-form-item>
            <el-form-item label="确认新密码" prop="confirmPassword">
              <el-input v-model="pwdForm.confirmPassword" type="password" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleChangePwd" :loading="pwdLoading">修改密码</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- About Card -->
      <el-col :span="12" :xs="24" style="margin-bottom: 20px">
        <el-card shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <el-icon :size="18"><InfoFilled /></el-icon>
              <span>关于系统</span>
            </div>
          </template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="系统名称">IPTV Tool</el-descriptions-item>
            <el-descriptions-item label="技术栈">Vue 3 + Element Plus / Go + Gin + SQLite</el-descriptions-item>
            <el-descriptions-item label="运行模式">单文件部署 (go:embed)</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { Lock, InfoFilled } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const pwdFormRef = ref()
const pwdLoading = ref(false)

const pwdForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const pwdRules = {
  oldPassword: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  newPassword: [{ required: true, message: '请输入新密码', trigger: 'blur' }, { min: 6, message: '至少6个字符', trigger: 'blur' }],
  confirmPassword: [{
    required: true, message: '请确认新密码', trigger: 'blur',
  }, {
    validator: (_, val, cb) => val === pwdForm.newPassword ? cb() : cb(new Error('两次密码不一致')),
    trigger: 'blur',
  }],
}

async function handleChangePwd() {
  await pwdFormRef.value.validate()
  pwdLoading.value = true
  try {
    await auth.changePassword(pwdForm.oldPassword, pwdForm.newPassword)
    ElMessage.success('密码修改成功')
    pwdForm.oldPassword = ''
    pwdForm.newPassword = ''
    pwdForm.confirmPassword = ''
  } catch { /* handled by interceptor */ }
  finally { pwdLoading.value = false }
}
</script>
