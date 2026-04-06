<template>
  <div>
    <h3 style="margin: 0 0 16px">{{ $t('settings_https.title') }}</h3>

    <!-- HTTPS Enable Card -->
    <el-card shadow="never" class="https-card">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon :size="18"><Connection /></el-icon>
            <span>{{ $t('settings_https.enable') }}</span>
          </div>
        </div>
      </template>

      <el-form :model="configForm" label-width="140px">
        <el-form-item :label="$t('settings_https.enable')">
          <div style="display: flex; align-items: center; gap: 12px">
            <el-switch v-model="configForm.enabled" />
            <el-text type="info" size="small">{{ $t('settings_https.enable_desc') }}</el-text>
          </div>
        </el-form-item>
        <el-form-item v-if="configForm.enabled" :label="$t('settings_https.port')">
          <el-input-number v-model="configForm.port" :min="1" :max="65535" :step="1" controls-position="right" style="width: 160px" />
          <span class="form-hint" style="margin-left: 12px">{{ $t('settings_https.port_help') }}</span>
          <div v-if="httpPort" style="width: 100%; margin-top: 4px">
            <el-text :type="configForm.port === httpPort ? 'danger' : 'info'" size="small">
              {{ $t('settings_https.port_current_http', { port: httpPort }) }}
            </el-text>
          </div>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- Certificate Management Card -->
    <el-card v-if="configForm.enabled" shadow="never" class="https-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon :size="18"><Key /></el-icon>
            <span>{{ $t('settings_https.cert_section') }}</span>
          </div>
        </div>
      </template>

      <el-text type="info" size="small" style="display: block; margin-bottom: 16px">
        {{ $t('settings_https.cert_section_desc') }}
      </el-text>

      <!-- Server Certificate -->
      <div class="cert-row">
        <div class="cert-info">
          <span class="cert-label">{{ $t('settings_https.server_cert') }}</span>
          <el-tag v-if="hasCert" type="success" size="small" effect="plain">
            <el-icon style="margin-right: 2px"><SuccessFilled /></el-icon>
            {{ $t('settings_https.cert_uploaded') }}
          </el-tag>
          <el-tag v-else type="info" size="small" effect="plain">
            {{ $t('settings_https.cert_not_uploaded') }}
          </el-tag>
        </div>
        <el-upload
          :action="uploadCertUrl"
          :headers="uploadHeaders"
          :show-file-list="false"
          :on-success="onCertUploadSuccess"
          :on-error="onUploadError"
          :before-upload="() => { uploadingCert = true; return true }"
          accept=".pem,.crt,.cer,.key"
        >
          <el-button type="primary" size="small" :loading="uploadingCert" :icon="Upload">
            {{ hasCert ? $t('settings_https.replace_cert') : $t('settings_https.upload_cert') }}
          </el-button>
        </el-upload>
      </div>

      <!-- Server Key -->
      <div class="cert-row">
        <div class="cert-info">
          <span class="cert-label">{{ $t('settings_https.server_key') }}</span>
          <el-tag v-if="hasKey" type="success" size="small" effect="plain">
            <el-icon style="margin-right: 2px"><SuccessFilled /></el-icon>
            {{ $t('settings_https.cert_uploaded') }}
          </el-tag>
          <el-tag v-else type="info" size="small" effect="plain">
            {{ $t('settings_https.cert_not_uploaded') }}
          </el-tag>
        </div>
        <el-upload
          :action="uploadKeyUrl"
          :headers="uploadHeaders"
          :show-file-list="false"
          :on-success="onKeyUploadSuccess"
          :on-error="onUploadError"
          :before-upload="() => { uploadingKey = true; return true }"
          accept=".pem,.crt,.cer,.key"
        >
          <el-button type="primary" size="small" :loading="uploadingKey" :icon="Upload">
            {{ hasKey ? $t('settings_https.replace_key') : $t('settings_https.upload_key') }}
          </el-button>
        </el-upload>
      </div>

      <el-text type="info" size="small" class="file-type-hint">
        {{ $t('settings_https.file_type_hint') }}
      </el-text>
    </el-card>

    <!-- Mutual TLS Card -->
    <el-card v-if="configForm.enabled" shadow="never" class="https-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon :size="18"><Lock /></el-icon>
            <span>{{ $t('settings_https.mutual_tls_section') }}</span>
          </div>
        </div>
      </template>

      <el-form :model="configForm" label-width="140px">
        <el-form-item :label="$t('settings_https.mutual_tls')">
          <div style="display: flex; align-items: center; gap: 12px">
            <el-switch v-model="configForm.mutualTLS" />
            <el-text type="info" size="small">{{ $t('settings_https.mutual_tls_desc') }}</el-text>
          </div>
        </el-form-item>
      </el-form>

      <!-- CA Certificate -->
      <div v-if="configForm.mutualTLS" class="cert-row" style="margin-top: 8px">
        <div class="cert-info">
          <span class="cert-label">{{ $t('settings_https.ca_cert') }}</span>
          <el-tag v-if="hasCACert" type="success" size="small" effect="plain">
            <el-icon style="margin-right: 2px"><SuccessFilled /></el-icon>
            {{ $t('settings_https.cert_uploaded') }}
          </el-tag>
          <el-tag v-else type="info" size="small" effect="plain">
            {{ $t('settings_https.cert_not_uploaded') }}
          </el-tag>
        </div>
        <div style="display: flex; gap: 8px">
          <el-upload
            :action="uploadCAUrl"
            :headers="uploadHeaders"
            :show-file-list="false"
            :on-success="onCAUploadSuccess"
            :on-error="onUploadError"
            :before-upload="() => { uploadingCA = true; return true }"
            accept=".pem,.crt,.cer"
          >
            <el-button type="primary" size="small" :loading="uploadingCA" :icon="Upload">
              {{ hasCACert ? $t('settings_https.replace_ca') : $t('settings_https.upload_ca') }}
            </el-button>
          </el-upload>
          <el-popconfirm
            v-if="hasCACert"
            :title="$t('settings_https.delete_ca_confirm')"
            :confirm-button-text="$t('common.confirm')"
            :cancel-button-text="$t('common.cancel')"
            @confirm="deleteCA"
          >
            <template #reference>
              <el-button type="danger" size="small" plain :icon="Delete">
                {{ $t('settings_https.delete_ca') }}
              </el-button>
            </template>
          </el-popconfirm>
        </div>
      </div>
    </el-card>

    <!-- Save Button -->
    <div style="margin-top: 20px; max-width: 700px">
      <el-button type="primary" @click="saveConfig" :loading="saving" size="large">
        {{ $t('common.save_config') }}
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Connection, Key, Lock, Upload, Delete, SuccessFilled } from '@element-plus/icons-vue'
import api from '../api'

const { t } = useI18n()

const configForm = reactive({
  enabled: false,
  port: 8024,
  mutualTLS: false,
})

const hasCert = ref(false)
const hasKey = ref(false)
const hasCACert = ref(false)
const httpPort = ref(0)
const saving = ref(false)
const uploadingCert = ref(false)
const uploadingKey = ref(false)
const uploadingCA = ref(false)

const uploadCertUrl = '/api/settings/https/cert'
const uploadKeyUrl = '/api/settings/https/key'
const uploadCAUrl = '/api/settings/https/ca'

const uploadHeaders = computed(() => {
  const token = localStorage.getItem('token')
  return token ? { Authorization: `Bearer ${token}` } : {}
})

onMounted(async () => {
  await loadSettings()
})

async function loadSettings() {
  try {
    const { data } = await api.get('/settings/https')
    configForm.enabled = data.enabled || false
    configForm.port = data.port || 8024
    configForm.mutualTLS = data.mutual_tls || false
    hasCert.value = data.has_cert || false
    hasKey.value = data.has_key || false
    hasCACert.value = data.has_ca_cert || false
    httpPort.value = data.http_port || 0
  } catch {}
}

function onCertUploadSuccess(response) {
  uploadingCert.value = false
  hasCert.value = true
  ElMessage.success(response.message || t('settings_https.upload_success'))
}

function onKeyUploadSuccess(response) {
  uploadingKey.value = false
  hasKey.value = true
  ElMessage.success(response.message || t('settings_https.upload_success'))
}

function onCAUploadSuccess(response) {
  uploadingCA.value = false
  hasCACert.value = true
  ElMessage.success(response.message || t('settings_https.upload_success'))
}

function onUploadError(error) {
  uploadingCert.value = false
  uploadingKey.value = false
  uploadingCA.value = false
  try {
    const resp = JSON.parse(error.message)
    ElMessage.error(resp.error || t('common.upload_failed'))
  } catch {
    ElMessage.error(t('common.upload_failed'))
  }
}

async function deleteCA() {
  try {
    await api.delete('/settings/https/ca')
    hasCACert.value = false
    ElMessage.success(t('settings_https.config_saved'))
  } catch {}
}

async function saveConfig() {
  // Client-side port conflict check
  if (configForm.enabled && httpPort.value > 0 && configForm.port === httpPort.value) {
    ElMessage.error(t('settings_https.port_current_http', { port: httpPort.value }))
    return
  }
  saving.value = true
  try {
    await api.put('/settings/https', {
      enabled: configForm.enabled,
      port: configForm.port,
      mutual_tls: configForm.mutualTLS,
    })
    ElMessage.success(t('settings_https.config_saved'))
  } catch {}
  finally { saving.value = false }
}
</script>

<style scoped>
.https-card {
  max-width: 700px;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.cert-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  margin-bottom: 10px;
  transition: background 0.2s;
}
.cert-row:hover {
  background: var(--el-fill-color-light);
}
.cert-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.cert-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  min-width: 100px;
}
.file-type-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
}
</style>
