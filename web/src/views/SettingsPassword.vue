<template>
  <div class="settings-page">
    <h3 style="margin: 0 0 20px">{{ $t('settings_password.title') }}</h3>

    <el-card shadow="hover" class="settings-card">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <el-icon :size="18"><Lock /></el-icon>
          <span>{{ $t('settings_password.title') }}</span>
        </div>
      </template>
      <el-form :model="pwdForm" :rules="pwdRules" ref="pwdFormRef" label-width="auto">
        <el-form-item :label="$t('settings_password.current_password')" prop="oldPassword">
          <el-input v-model="pwdForm.oldPassword" type="password" show-password />
        </el-form-item>
        <el-form-item :label="$t('settings_password.new_password')" prop="newPassword">
          <el-input v-model="pwdForm.newPassword" type="password" show-password />
        </el-form-item>
        <el-form-item :label="$t('settings_password.confirm_password')" prop="confirmPassword">
          <el-input v-model="pwdForm.confirmPassword" type="password" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleChangePwd" :loading="pwdLoading">{{ $t('settings_password.change_btn') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Lock } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'

const { t } = useI18n()

const auth = useAuthStore()
const pwdFormRef = ref()
const pwdLoading = ref(false)

const pwdForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const pwdRules = computed(() => ({
  oldPassword: [{ required: true, message: t('settings_password.current_password_required'), trigger: 'blur' }],
  newPassword: [{ required: true, message: t('settings_password.new_password_required'), trigger: 'blur' }, { min: 6, message: t('settings_password.min_length'), trigger: 'blur' }],
  confirmPassword: [{
    required: true, message: t('settings_password.confirm_password_required'), trigger: 'blur',
  }, {
    validator: (_, val, cb) => val === pwdForm.newPassword ? cb() : cb(new Error(t('settings_password.password_mismatch'))),
    trigger: 'blur',
  }],
}))

async function handleChangePwd() {
  await pwdFormRef.value.validate()
  pwdLoading.value = true
  try {
    await auth.changePassword(pwdForm.oldPassword, pwdForm.newPassword)
    ElMessage.success(t('settings_password.change_success'))
    pwdForm.oldPassword = ''
    pwdForm.newPassword = ''
    pwdForm.confirmPassword = ''
  } catch { /* handled by interceptor */ }
  finally { pwdLoading.value = false }
}
</script>

<style scoped>
.settings-page {
  width: 100%;
}
.settings-card {
  max-width: 600px;
}
</style>
