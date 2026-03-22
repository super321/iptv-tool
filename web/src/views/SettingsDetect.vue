<template>
  <div>
    <h3 style="margin: 0 0 16px">{{ $t('settings_detect.title') }}</h3>

    <el-card shadow="never" style="max-width: 700px">
      <template #header>
        <span>{{ $t('settings_detect.ffprobe_section') }}</span>
      </template>

      <el-descriptions :column="1" border>
        <el-descriptions-item :label="$t('settings_detect.current_version')">
          <span v-if="ffprobeVersion" style="color: var(--el-color-success)">
            {{ ffprobeVersion }}
            <el-tag v-if="ffprobeSource === 'uploaded'" size="small" type="success" style="margin-left: 8px">{{ $t('settings_detect.user_uploaded') }}</el-tag>
            <el-tag v-else-if="ffprobeSource === 'system'" size="small" type="info" style="margin-left: 8px">{{ $t('settings_detect.system_builtin') }}</el-tag>
          </span>
          <span v-else style="color: #909399">{{ $t('settings_detect.not_configured') }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <div style="margin-top: 16px">
        <el-upload
          ref="uploadRef"
          :action="uploadUrl"
          :headers="uploadHeaders"
          :show-file-list="false"
          :on-success="onUploadSuccess"
          :on-error="onUploadError"
          :before-upload="beforeUpload"
        >
          <el-button type="primary" :loading="uploading">
            {{ ffprobeVersion ? $t('settings_detect.update_ffprobe') : $t('settings_detect.upload_ffprobe') }}
          </el-button>
        </el-upload>
        <div style="color: #909399; font-size: 12px; margin-top: 8px; line-height: 1.6">
          {{ $t('settings_detect.upload_help') }}<br/>
          {{ $t('settings_detect.ffmpeg_download_prefix') }} <a href="https://ffmpeg.org/download.html" target="_blank" rel="noopener noreferrer" style="color: var(--el-color-primary); text-decoration: none;">{{ $t('settings_detect.ffmpeg_download_link') }}</a>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" style="max-width: 700px; margin-top: 16px">
      <template #header>
        <span>{{ $t('settings_detect.params_section') }}</span>
      </template>

      <el-form :model="configForm" label-width="120px">
        <el-form-item :label="$t('settings_detect.concurrency')">
          <el-input-number v-model="configForm.concurrency" :min="1" :max="30" />
          <span style="margin-left: 12px; color: #909399; font-size: 12px">{{ $t('settings_detect.concurrency_help') }}</span>
        </el-form-item>
        <el-form-item :label="$t('settings_detect.timeout')">
          <el-input-number v-model="configForm.timeout" :min="1" :max="30" />
          <span style="margin-left: 12px; color: #909399; font-size: 12px">{{ $t('settings_detect.timeout_help') }}</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveConfig" :loading="saving">{{ $t('settings_detect.save_config') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import api from '../api'

const { t } = useI18n()

const ffprobeVersion = ref('')
const ffprobeSource = ref('')
const uploading = ref(false)
const saving = ref(false)
const uploadRef = ref()

const configForm = reactive({
  concurrency: 10,
  timeout: 5,
})

const uploadUrl = '/api/settings/detect/ffprobe'
const uploadHeaders = computed(() => {
  const token = localStorage.getItem('token')
  return token ? { Authorization: `Bearer ${token}` } : {}
})

onMounted(async () => {
  await loadSettings()
})

async function loadSettings() {
  try {
    const { data } = await api.get('/settings/detect')
    configForm.concurrency = data.concurrency || 10
    configForm.timeout = data.timeout || 5
    ffprobeVersion.value = data.ffprobe_version || ''
    ffprobeSource.value = data.ffprobe_source || ''
  } catch {}
}

function beforeUpload() {
  uploading.value = true
  return true
}

function onUploadSuccess(response) {
  uploading.value = false
  if (response.ffprobe_version) {
    ffprobeVersion.value = response.ffprobe_version
    ffprobeSource.value = response.ffprobe_source || 'uploaded'
  }
  ElMessage.success(response.message || t('settings_detect.upload_success'))
}

function onUploadError(error) {
  uploading.value = false
  try {
    const resp = JSON.parse(error.message)
    ElMessage.error(resp.error || t('settings_detect.upload_failed'))
  } catch {
    ElMessage.error(t('settings_detect.upload_failed'))
  }
}

async function saveConfig() {
  saving.value = true
  try {
    await api.put('/settings/detect', {
      concurrency: configForm.concurrency,
      timeout: configForm.timeout,
    })
    ElMessage.success(t('settings_detect.config_saved'))
  } catch {}
  finally { saving.value = false }
}
</script>
