<template>
  <div>
    <h3 style="margin: 0 0 16px">直播检测设置</h3>

    <el-card shadow="never" style="max-width: 700px">
      <template #header>
        <span>ffprobe 可执行文件</span>
      </template>

      <el-descriptions :column="1" border>
        <el-descriptions-item label="当前版本">
          <span v-if="ffprobeVersion" style="color: #67c23a">{{ ffprobeVersion }}</span>
          <span v-else style="color: #909399">未上传</span>
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
            {{ ffprobeVersion ? '更新 ffprobe' : '上传 ffprobe' }}
          </el-button>
        </el-upload>
        <div style="color: #909399; font-size: 12px; margin-top: 8px; line-height: 1.6">
          请上传 ffprobe 可执行文件。上传后系统将自动验证文件有效性。<br/>
          ffprobe 包含在 FFmpeg 发行包中，如需下载请前往 <a href="https://ffmpeg.org/download.html" target="_blank" rel="noopener noreferrer" style="color: var(--el-color-primary); text-decoration: none;">ffmpeg 官网下载页面</a>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" style="max-width: 700px; margin-top: 16px">
      <template #header>
        <span>检测参数</span>
      </template>

      <el-form :model="configForm" label-width="120px">
        <el-form-item label="检测并发数">
          <el-input-number v-model="configForm.concurrency" :min="1" :max="30" />
          <span style="margin-left: 12px; color: #909399; font-size: 12px">范围 1-30，默认 10</span>
        </el-form-item>
        <el-form-item label="检测超时">
          <el-input-number v-model="configForm.timeout" :min="1" :max="30" />
          <span style="margin-left: 12px; color: #909399; font-size: 12px">范围 1-30 秒，默认 5 秒</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveConfig" :loading="saving">保存配置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const ffprobeVersion = ref('')
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
  }
  ElMessage.success(response.message || 'ffprobe 上传成功')
}

function onUploadError(error) {
  uploading.value = false
  try {
    const resp = JSON.parse(error.message)
    ElMessage.error(resp.error || '上传失败')
  } catch {
    ElMessage.error('上传失败')
  }
}

async function saveConfig() {
  saving.value = true
  try {
    await api.put('/settings/detect', {
      concurrency: configForm.concurrency,
      timeout: configForm.timeout,
    })
    ElMessage.success('配置已保存')
  } catch {}
  finally { saving.value = false }
}
</script>
