<template>
  <div class="log-container">
    <div class="log-header">
      <h2>{{ $t('logs.access_title') }}</h2>
      <div class="log-actions">
        <el-input
            v-model="ipFilter"
            :placeholder="$t('logs.filter_ip')"
            :prefix-icon="Search"
            size="small"
            clearable
            style="width: 180px"
        />
        <el-input
            v-model="pathFilter"
            :placeholder="$t('logs.filter_path')"
            :prefix-icon="Search"
            size="small"
            clearable
            style="width: 220px"
        />
        <el-button
            :type="isPaused ? 'success' : 'warning'"
            @click="isPaused = !isPaused"
            :icon="isPaused ? VideoPlay : VideoPause"
            size="small"
        >
          {{ isPaused ? $t('logs.resume') : $t('logs.pause') }}
        </el-button>
        <el-button type="danger" :icon="Delete" size="small" plain @click="confirmClear">
          {{ $t('logs.clear') }}
        </el-button>
        <el-button type="primary" :icon="Download" size="small" plain @click="downloadLogs">
          {{ $t('logs.download') }}
        </el-button>
      </div>
    </div>
    <div class="log-table-wrapper">
      <el-table
          :data="filteredEntries"
          stripe
          size="small"
          :empty-text="$t('logs.no_logs')"
          class="access-table"
          height="100%"
      >
        <el-table-column prop="time" :label="$t('logs.col_time')" width="170" />
        <el-table-column prop="client_ip" :label="$t('logs.col_ip')" width="150" />
        <el-table-column :label="$t('logs.col_request')" min-width="280">
          <template #default="{ row }">
            <span :class="['method-tag', `method-${row.method.toLowerCase()}`]">{{ row.method }}</span>
            <span class="path-text">{{ row.path }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="$t('logs.col_status')" width="100" align="center">
          <template #default="{ row }">
            <el-tag
                :type="statusType(row.status)"
                size="small"
                effect="dark"
                round
            >{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="latency" :label="$t('logs.col_latency')" width="120" align="right" />
        <el-table-column prop="user_agent" :label="$t('logs.col_ua')" min-width="200" show-overflow-tooltip />
      </el-table>
    </div>
    <div class="log-status-bar">
      <span>{{ $t('logs.total_lines', { count: filteredEntries.length }) }}</span>
      <span :class="['status-dot', isPaused ? 'paused' : 'live']"></span>
      <span>{{ isPaused ? $t('logs.status_paused') : $t('logs.status_live') }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Delete, Download, VideoPlay, VideoPause, Search } from '@element-plus/icons-vue'
import api from '../api'

const { t } = useI18n()
const entries = ref([])
const isPaused = ref(false)
const lastID = ref(0)
const ipFilter = ref('')
const pathFilter = ref('')
let timer = null

const filteredEntries = computed(() => {
  const ip = ipFilter.value.toLowerCase()
  const path = pathFilter.value.toLowerCase()
  if (!ip && !path) return entries.value
  return entries.value.filter(e => {
    if (ip && !e.client_ip.toLowerCase().includes(ip)) return false
    if (path && !e.path.toLowerCase().includes(path)) return false
    return true
  })
})

function statusType(status) {
  if (status >= 200 && status < 300) return 'success'
  if (status >= 300 && status < 400) return 'warning'
  if (status >= 400 && status < 500) return 'danger'
  if (status >= 500) return 'danger'
  return 'info'
}

async function fetchLogs() {
  if (isPaused.value) return
  try {
    const { data } = await api.get(`/logs/access?since=${lastID.value}`)
    if (data.entries && data.entries.length > 0) {
      // API returns newest-first; find max ID for next poll
      let maxID = lastID.value
      for (const e of data.entries) {
        if (e.id > maxID) maxID = e.id
      }
      lastID.value = maxID
      // Prepend new entries at the top (they are already newest-first)
      entries.value.unshift(...data.entries)
      // Keep max 10000 entries, trim oldest from the end
      if (entries.value.length > 10000) {
        entries.value = entries.value.slice(0, 5000)
      }
    }
  } catch {
    // Silently ignore polling errors
  }
}

function confirmClear() {
  ElMessageBox.confirm(t('logs.clear_confirm'), t('logs.clear'), {
    confirmButtonText: t('common.confirm'),
    cancelButtonText: t('common.cancel'),
    type: 'warning',
  }).then(() => clearLogs()).catch(() => {})
}

async function clearLogs() {
  try {
    await api.delete('/logs/access')
    entries.value = []
    lastID.value = 0
    ElMessage.success(t('logs.clear_success'))
  } catch {
    // error handled by interceptor
  }
}

async function downloadLogs() {
  try {
    const response = await api.get('/logs/access/download', { responseType: 'blob' })
    const url = URL.createObjectURL(response.data)
    const a = document.createElement('a')
    a.href = url
    a.download = response.headers['content-disposition']?.split('filename=')[1] || 'access.log'
    a.click()
    URL.revokeObjectURL(url)
  } catch {
    // error handled by interceptor
  }
}

onMounted(() => {
  fetchLogs()
  timer = setInterval(fetchLogs, 2000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.log-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 108px);
  background: var(--el-bg-color-overlay);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}
.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: var(--log-access-header-bg);
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}
.log-header h2 {
  color: var(--el-text-color-primary);
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}
.log-actions {
  display: flex;
  gap: 8px;
}
.log-table-wrapper {
  flex: 1;
  overflow: hidden;
}
.method-tag {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 600;
  font-family: monospace;
  margin-right: 6px;
  min-width: 50px;
  text-align: center;
}
.method-get { background: var(--method-get-bg); color: var(--method-get-color); }
.method-post { background: var(--method-post-bg); color: var(--method-post-color); }
.method-put { background: var(--method-put-bg); color: var(--method-put-color); }
.method-delete { background: var(--method-delete-bg); color: var(--method-delete-color); }
.method-patch { background: var(--method-patch-bg); color: var(--method-patch-color); }
.path-text {
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.log-status-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 20px;
  background: #007acc;
  color: #fff;
  font-size: 12px;
  flex-shrink: 0;
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}
.status-dot.live {
  background: #4ec9b0;
  animation: pulse 1.5s infinite;
}
.status-dot.paused {
  background: #ce9178;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
