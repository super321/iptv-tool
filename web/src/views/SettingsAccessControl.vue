<template>
  <div class="acl-page">
    <h3 style="margin: 0 0 20px">{{ $t('settings_access_control.title') }}</h3>

    <!-- GeoIP Database Card -->
    <el-card shadow="hover" class="acl-card">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon :size="18"><Location /></el-icon>
            <span>{{ $t('settings_access_control.geoip_title') }}</span>
          </div>
          <el-button
            type="primary"
            size="small"
            :icon="Refresh"
            :loading="geoipDownloading"
            @click="checkGeoIPUpdate"
          >
            {{ geoipDownloading ? $t('settings_access_control.geoip_downloading') : $t('settings_access_control.geoip_check_update') }}
          </el-button>
        </div>
      </template>

      <el-descriptions :column="2" border size="small">
        <el-descriptions-item :label="$t('settings_access_control.geoip_version')">
          <!-- Show download progress during download -->
          <template v-if="geoipDownloading">
            <el-text type="primary" size="small">
              <template v-if="downloadProgress.percent">{{ $t('settings_access_control.geoip_downloading_progress', { percent: downloadProgress.percent }) }}</template>
              <template v-else-if="downloadProgress.downloaded_bytes > 0">{{ $t('settings_access_control.geoip_downloading_no_size', { downloaded: formatFileSize(downloadProgress.downloaded_bytes) }) }}</template>
              <template v-else>{{ $t('settings_access_control.geoip_downloading') }}</template>
            </el-text>
            <el-text v-if="downloadProgress.attempt > 1" type="warning" size="small" style="margin-left: 8px">
              {{ $t('settings_access_control.geoip_download_attempt', { attempt: downloadProgress.attempt, max: downloadProgress.max_retries }) }}
            </el-text>
          </template>
          <!-- Show version or not-downloaded -->
          <template v-else>
            <el-tag v-if="geoipStatus.exists" type="success" size="small" effect="plain">{{ geoipStatus.version }}</el-tag>
            <el-tag v-else type="info" size="small" effect="plain">{{ $t('settings_access_control.geoip_not_downloaded') }}</el-tag>
          </template>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('settings_access_control.geoip_auto_update')">
          <div style="display: flex; align-items: center; gap: 12px">
            <el-switch v-model="geoipAutoUpdate" @change="saveGeoIPAutoUpdate" />
            <template v-if="geoipAutoUpdate">
              <span style="color: var(--el-text-color-regular); font-size: 12px">{{ $t('settings_access_control.geoip_update_interval_label') }}</span>
              <el-input-number
                v-model="geoipIntervalDays"
                :min="1"
                :max="7"
                size="small"
                style="width: 100px"
                @change="saveGeoIPAutoUpdate"
              />
              <span style="color: #909399; font-size: 12px">{{ $t('settings_access_control.geoip_interval_days') }}</span>
            </template>
          </div>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- Access Statistics Card -->
    <el-card shadow="hover" class="acl-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon :size="18"><DataAnalysis /></el-icon>
            <span>{{ $t('settings_access_control.access_stats_title') }}</span>
            <el-text type="info" size="small">{{ $t('settings_access_control.access_stats_desc') }}</el-text>
          </div>
          <el-button size="small" :icon="Refresh" :loading="accessStatsLoading" @click="loadAccessStats">
            {{ $t('settings_access_control.access_stats_refresh') }}
          </el-button>
        </div>
      </template>

      <el-table :data="accessStats" stripe style="width: 100%" size="small" v-loading="accessStatsLoading">
        <el-table-column prop="ip" :label="$t('settings_access_control.col_access_ip')" min-width="180">
          <template #default="{ row }">
            <code class="ip-value">{{ row.ip }}</code>
          </template>
        </el-table-column>
        <el-table-column min-width="160">
          <template #header>
            <div style="display: flex; align-items: center; gap: 4px">
              <span>{{ $t('settings_access_control.col_access_location') }}</span>
              <el-tooltip :content="$t('settings_access_control.access_location_tip')" placement="top" :show-after="300">
                <el-icon :size="14" style="color: #909399; cursor: help"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </template>
          <template #default="{ row }">
            <span v-if="row.location">{{ row.location }}</span>
            <span v-else style="color: var(--el-text-color-placeholder)">—</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('settings_access_control.col_access_last_time')" min-width="170">
          <template #default="{ row }">
            {{ formatTime(row.last_accessed_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="total_requests" :label="$t('settings_access_control.col_access_total')" width="120" align="center" />
        <el-table-column prop="sub_requests" :label="$t('settings_access_control.col_access_sub')" width="150" align="center" />
      </el-table>
      <div v-if="accessStatsTotal > 0" style="margin-top: 12px; display: flex; justify-content: flex-end">
        <el-pagination
          v-model:current-page="accessStatsPage"
          :page-size="accessStatsPageSize"
          :total="accessStatsTotal"
          layout="total, prev, pager, next"
          small
          @current-change="loadAccessStats"
        />
      </div>
      <el-empty v-if="!accessStatsLoading && accessStats.length === 0" :description="$t('settings_access_control.access_no_data')" :image-size="60" />
    </el-card>

    <!-- Access Control Mode Card -->
    <el-card shadow="hover" class="acl-card" style="margin-top: 16px">
      <!-- Mode Selection -->
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon :size="18"><Warning /></el-icon>
            <span>{{ $t('settings_access_control.mode_label') }}</span>
          </div>
        </div>
      </template>

      <div class="mode-group">
        <el-radio-group v-model="mode" @change="onModeChange" class="mode-radio-group">
          <!-- Disabled Mode -->
          <div :class="['mode-card', { active: mode === 'disabled' }]" @click="mode = 'disabled'; onModeChange()">
            <div class="mode-card-header">
              <el-radio value="disabled" class="mode-radio" />
              <span class="mode-name">{{ $t('settings_access_control.mode_disabled') }}</span>
            </div>
            <div class="mode-card-divider"></div>
            <div class="mode-card-body">
              <el-icon :size="80" class="mode-icon disabled-icon"><CircleClose /></el-icon>
              <span class="mode-hint">{{ $t('settings_access_control.mode_disabled_desc') }}</span>
            </div>
          </div>

          <!-- Whitelist Mode -->
          <div :class="['mode-card', { active: mode === 'whitelist' }]" @click="mode = 'whitelist'; onModeChange()">
            <div class="mode-card-header">
              <el-radio value="whitelist" class="mode-radio" />
              <span class="mode-name">{{ $t('settings_access_control.mode_whitelist') }}</span>
            </div>
            <div class="mode-card-divider"></div>
            <div class="mode-card-body">
              <el-icon :size="80" class="mode-icon whitelist-icon"><CircleCheck /></el-icon>
              <span class="mode-hint">{{ $t('settings_access_control.mode_whitelist_desc') }}</span>
            </div>
          </div>

          <!-- Blacklist Mode -->
          <div :class="['mode-card', { active: mode === 'blacklist' }]" @click="mode = 'blacklist'; onModeChange()">
            <div class="mode-card-header">
              <el-radio value="blacklist" class="mode-radio" />
              <span class="mode-name">{{ $t('settings_access_control.mode_blacklist') }}</span>
            </div>
            <div class="mode-card-divider"></div>
            <div class="mode-card-body">
              <el-icon :size="80" class="mode-icon blacklist-icon"><Remove /></el-icon>
              <span class="mode-hint">{{ $t('settings_access_control.mode_blacklist_desc') }}</span>
            </div>
          </div>
        </el-radio-group>
      </div>
    </el-card>

    <!-- Whitelist Management -->
    <el-card v-if="mode === 'whitelist'" shadow="hover" class="acl-card list-card">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon :size="18"><CircleCheck /></el-icon>
            <span>{{ $t('settings_access_control.whitelist_title') }}</span>
            <el-tag size="small" type="success" effect="plain">{{ entries.length }}</el-tag>
          </div>
          <el-button type="primary" size="small" :icon="Plus" @click="showAddDialog('whitelist')">
            {{ $t('settings_access_control.add_entry') }}
          </el-button>
        </div>
      </template>

      <el-table :data="pagedWhitelistEntries" stripe style="width: 100%" v-if="entries.length > 0" size="small">
        <el-table-column prop="value" :label="$t('settings_access_control.col_ip')" min-width="220">
          <template #default="{ row }">
            <code class="ip-value">{{ row.value }}</code>
          </template>
        </el-table-column>
        <el-table-column :label="$t('settings_access_control.col_type')" width="140">
          <template #default="{ row }">
            <el-tag size="small" :type="entryTypeTagType(row.entry_type)" effect="plain">
              {{ entryTypeLabel(row.entry_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.operations')" width="80" align="center">
          <template #default="{ row }">
            <el-button type="danger" link size="small" :icon="Delete" @click="removeEntry(entries.indexOf(row))" />
          </template>
        </el-table-column>
      </el-table>
      <div v-if="entries.length > aclPageSize" style="margin-top: 12px; display: flex; justify-content: flex-end">
        <el-pagination
          v-model:current-page="whitelistPage"
          :page-size="aclPageSize"
          :total="entries.length"
          layout="total, prev, pager, next"
          small
        />
      </div>
      <el-empty v-if="entries.length === 0" :description="$t('settings_access_control.no_entries')" :image-size="60" />
    </el-card>

    <!-- Blacklist Management -->
    <el-card v-if="mode === 'blacklist'" shadow="hover" class="acl-card list-card">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon :size="18"><Remove /></el-icon>
            <span>{{ $t('settings_access_control.blacklist_title') }}</span>
            <el-tag size="small" type="danger" effect="plain">{{ entries.length }}</el-tag>
          </div>
          <el-button type="primary" size="small" :icon="Plus" @click="showAddDialog('blacklist')">
            {{ $t('settings_access_control.add_entry') }}
          </el-button>
        </div>
      </template>

      <el-table :data="pagedBlacklistEntries" stripe style="width: 100%" v-if="entries.length > 0" size="small">
        <el-table-column :label="$t('settings_access_control.col_ip')" min-width="200">
          <template #default="{ row }">
            <code class="ip-value">{{ row.value }}</code>
          </template>
        </el-table-column>
        <el-table-column :label="$t('settings_access_control.col_block_type')" width="130">
          <template #default="{ row }">
            <el-tag v-if="row.block_days == null || row.block_days === 0" type="danger" size="small" effect="plain">
              {{ $t('settings_access_control.block_permanent') }}
            </el-tag>
            <el-tag v-else type="warning" size="small" effect="plain">
              {{ $t('settings_access_control.block_days', { days: row.block_days }) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('settings_access_control.col_remaining')" width="140">
          <template #default="{ row }">
            <span v-if="row.block_days == null || row.block_days === 0" style="color: var(--el-color-danger); font-size: 12px">
              {{ $t('settings_access_control.block_permanent') }}
            </span>
            <span v-else :style="{ color: isExpired(row) ? 'var(--el-color-success)' : 'var(--el-color-warning)', fontSize: '12px' }">
              {{ computeRemaining(row) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.operations')" width="80" align="center">
          <template #default="{ row }">
            <el-button type="danger" link size="small" :icon="Delete" @click="removeEntry(entries.indexOf(row))" />
          </template>
        </el-table-column>
      </el-table>
      <div v-if="entries.length > aclPageSize" style="margin-top: 12px; display: flex; justify-content: flex-end">
        <el-pagination
          v-model:current-page="blacklistPage"
          :page-size="aclPageSize"
          :total="entries.length"
          layout="total, prev, pager, next"
          small
        />
      </div>
      <el-empty v-if="entries.length === 0" :description="$t('settings_access_control.no_entries')" :image-size="60" />
    </el-card>

    <!-- Save Button — always visible when anything has changed -->
    <div class="save-bar" v-if="hasChanges">
      <el-button type="primary" @click="saveSettings" :loading="saving" size="large">
        {{ $t('settings_access_control.save') }}
      </el-button>
    </div>

    <!-- Add Whitelist Entry Dialog -->
    <el-dialog v-model="addDialogVisible" :title="addDialogTitle" width="520px" @close="resetAddForm" align-center>
      <el-form :model="addForm" label-width="110px" label-position="right">
        <!-- Whitelist: entry type selector -->
        <el-form-item v-if="addListType === 'whitelist'" :label="$t('settings_access_control.entry_type')">
          <el-select v-model="addForm.entry_type" style="width: 100%" @change="addForm.value = ''; addForm.rangeStart = ''; addForm.rangeEnd = ''">
            <el-option value="single" :label="$t('settings_access_control.type_single')" />
            <el-option value="cidr" :label="$t('settings_access_control.type_cidr')" />
            <el-option value="range" :label="$t('settings_access_control.type_range')" />
          </el-select>
        </el-form-item>

        <!-- Single IP / CIDR input (shown for all types except range) -->
        <el-form-item v-if="addForm.entry_type !== 'range' || addListType === 'blacklist'" :label="addListType === 'blacklist' ? $t('settings_access_control.type_single') : $t('settings_access_control.col_ip')">
          <el-input v-model="addForm.value" :placeholder="valuePlaceholder" clearable />
        </el-form-item>

        <!-- IP Range: separate start and end inputs -->
        <template v-if="addForm.entry_type === 'range' && addListType === 'whitelist'">
          <el-form-item :label="$t('settings_access_control.range_start')">
            <el-input v-model="addForm.rangeStart" :placeholder="$t('settings_access_control.ip_placeholder')" clearable />
          </el-form-item>
          <el-form-item :label="$t('settings_access_control.range_end')">
            <el-input v-model="addForm.rangeEnd" :placeholder="$t('settings_access_control.ip_placeholder')" clearable />
          </el-form-item>
        </template>

        <!-- Blacklist: block type -->
        <el-form-item v-if="addListType === 'blacklist'" :label="$t('settings_access_control.block_type')">
          <el-radio-group v-model="addForm.blockType">
            <el-radio value="permanent">{{ $t('settings_access_control.block_type_permanent') }}</el-radio>
            <el-radio value="days">{{ $t('settings_access_control.block_type_days') }}</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="addListType === 'blacklist' && addForm.blockType === 'days'" :label="$t('settings_access_control.block_days_label')">
          <el-input-number v-model="addForm.blockDays" :min="1" :max="3650" style="width: 160px" />
          <span style="margin-left: 8px; color: #909399; font-size: 12px">{{ $t('settings_access_control.block_days_unit') }}</span>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="addDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmAddEntry">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Warning, CircleClose, CircleCheck, Remove, Plus, Delete, Location, Refresh, DataAnalysis, QuestionFilled } from '@element-plus/icons-vue'
import api from '../api'

const { t } = useI18n()

// --- GeoIP State ---
const geoipStatus = reactive({ exists: false, version: '' })
const geoipAutoUpdate = ref(false)
const geoipIntervalDays = ref(1)
const geoipDownloading = ref(false)
const downloadProgress = reactive({ percent: '', downloaded_bytes: 0, total_bytes: 0, attempt: 0, max_retries: 3, error: '' })
let progressTimer = null

// --- Access Stats State ---
const accessStats = ref([])
const accessStatsTotal = ref(0)
const accessStatsPage = ref(1)
const accessStatsPageSize = 15
const accessStatsLoading = ref(false)

// --- Access Control State ---
const mode = ref('disabled')
const originalMode = ref('disabled')
const entries = ref([])
const originalEntries = ref([])
const saving = ref(false)
const whitelistPage = ref(1)
const blacklistPage = ref(1)
const aclPageSize = 10

// Paginated entries for whitelist/blacklist
const pagedWhitelistEntries = computed(() => {
  const start = (whitelistPage.value - 1) * aclPageSize
  return entries.value.slice(start, start + aclPageSize)
})
const pagedBlacklistEntries = computed(() => {
  const start = (blacklistPage.value - 1) * aclPageSize
  return entries.value.slice(start, start + aclPageSize)
})

// Track whether there are unsaved changes
const hasChanges = computed(() => {
  if (mode.value !== originalMode.value) return true
  if (JSON.stringify(entries.value.map(e => ({ entry_type: e.entry_type, value: e.value, block_days: e.block_days })))
    !== JSON.stringify(originalEntries.value.map(e => ({ entry_type: e.entry_type, value: e.value, block_days: e.block_days })))) return true
  return false
})

// Add dialog state
const addDialogVisible = ref(false)
const addListType = ref('whitelist')
const addForm = reactive({
  entry_type: 'single',
  value: '',
  rangeStart: '',
  rangeEnd: '',
  blockType: 'permanent',
  blockDays: 7,
})

const addDialogTitle = computed(() => {
  return addListType.value === 'whitelist'
    ? t('settings_access_control.add_whitelist_title')
    : t('settings_access_control.add_blacklist_title')
})

const valuePlaceholder = computed(() => {
  if (addListType.value === 'blacklist') return t('settings_access_control.ip_placeholder')
  switch (addForm.entry_type) {
    case 'cidr': return t('settings_access_control.cidr_placeholder')
    default: return t('settings_access_control.ip_placeholder')
  }
})

function entryTypeLabel(type) {
  switch (type) {
    case 'single': return t('settings_access_control.type_single')
    case 'cidr': return t('settings_access_control.type_cidr')
    case 'range': return t('settings_access_control.type_range')
    default: return type
  }
}

function entryTypeTagType(type) {
  switch (type) {
    case 'single': return 'info'
    case 'cidr': return 'success'
    case 'range': return 'warning'
    default: return 'info'
  }
}

function isExpired(row) {
  if (row.block_days == null || row.block_days === 0) return false
  const created = new Date(row.created_at)
  const expiry = new Date(created.getTime() + row.block_days * 24 * 60 * 60 * 1000)
  return new Date() >= expiry
}

function computeRemaining(row) {
  if (row.block_days == null || row.block_days === 0) return t('settings_access_control.block_permanent')
  const created = new Date(row.created_at)
  const expiry = new Date(created.getTime() + row.block_days * 24 * 60 * 60 * 1000)
  const now = new Date()
  if (now >= expiry) return t('settings_access_control.block_expired')
  const diffMs = expiry - now
  const diffDays = Math.floor(diffMs / (24 * 60 * 60 * 1000))
  if (diffDays > 0) return t('settings_access_control.remaining_days', { days: diffDays })
  const diffHours = Math.ceil(diffMs / (60 * 60 * 1000))
  return t('settings_access_control.remaining_hours', { hours: diffHours })
}

onMounted(async () => {
  await Promise.all([loadGeoIPStatus(), checkInitialDownloadState(), loadAccessStats(), loadSettings()])
})

// Check if a download is already in progress when the page loads
async function checkInitialDownloadState() {
  try {
    const { data } = await api.get('/settings/geoip/progress')
    if (data.downloading) {
      geoipDownloading.value = true
      downloadProgress.percent = data.percent || ''
      downloadProgress.downloaded_bytes = data.downloaded_bytes || 0
      downloadProgress.total_bytes = data.total_bytes || 0
      downloadProgress.attempt = data.attempt || 0
      downloadProgress.max_retries = data.max_retries || 3
      startProgressPolling()
    }
  } catch {}
}

// --- GeoIP Methods ---
async function loadGeoIPStatus() {
  try {
    const { data } = await api.get('/settings/geoip/status')
    geoipStatus.exists = data.exists
    geoipStatus.version = data.version
    geoipAutoUpdate.value = data.auto_update
    geoipIntervalDays.value = data.update_interval_days || 1
  } catch {}
}

async function checkGeoIPUpdate() {
  // Prevent double-click
  if (geoipDownloading.value) {
    ElMessage.warning(t('settings_access_control.geoip_downloading_tip'))
    return
  }

  geoipDownloading.value = true
  try {
    await api.post('/settings/geoip/check-update')
    // Start polling for progress
    startProgressPolling()
  } catch {
    geoipDownloading.value = false
  }
}

function startProgressPolling() {
  stopProgressPolling()
  progressTimer = setInterval(async () => {
    try {
      const { data } = await api.get('/settings/geoip/progress')
      downloadProgress.percent = data.percent || ''
      downloadProgress.downloaded_bytes = data.downloaded_bytes || 0
      downloadProgress.total_bytes = data.total_bytes || 0
      downloadProgress.attempt = data.attempt || 0
      downloadProgress.max_retries = data.max_retries || 3
      downloadProgress.error = data.error || ''

      if (!data.downloading) {
        // Download finished
        stopProgressPolling()
        geoipDownloading.value = false

        if (data.error) {
          ElMessage.error(data.error)
        } else if (data.exists) {
          geoipStatus.exists = true
          geoipStatus.version = data.version
          ElMessage.success(t('settings_access_control.geoip_download_success'))
          await loadAccessStats()
        }
      }
    } catch {
      // Ignore polling errors
    }
  }, 2000)
}

function stopProgressPolling() {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

function formatFileSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

onBeforeUnmount(() => {
  stopProgressPolling()
})

async function saveGeoIPAutoUpdate() {
  try {
    await api.put('/settings/geoip/auto-update', {
      enabled: geoipAutoUpdate.value,
      interval_days: geoipIntervalDays.value,
    })
    ElMessage.success(t('settings_access_control.geoip_auto_update_saved'))
  } catch {}
}

// --- Access Stats Methods ---
async function loadAccessStats() {
  accessStatsLoading.value = true
  try {
    const { data } = await api.get('/settings/access-stats', {
      params: { page: accessStatsPage.value, page_size: accessStatsPageSize },
    })
    accessStats.value = data.items || []
    accessStatsTotal.value = data.total || 0
  } catch {} finally {
    accessStatsLoading.value = false
  }
}

function formatTime(isoStr) {
  if (!isoStr) return ''
  const d = new Date(isoStr)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

// --- Access Control Methods ---
async function loadSettings() {
  try {
    const { data } = await api.get('/settings/access-control')
    mode.value = data.mode || 'disabled'
    originalMode.value = data.mode || 'disabled'
    entries.value = data.entries || []
    originalEntries.value = JSON.parse(JSON.stringify(data.entries || []))
  } catch {}
}

function onModeChange() {
  // When switching modes, clear entries
  if (mode.value !== originalMode.value) {
    entries.value = []
  } else {
    // Restoring to original mode, restore original entries
    entries.value = JSON.parse(JSON.stringify(originalEntries.value))
  }
}

function showAddDialog(listType) {
  addListType.value = listType
  addForm.entry_type = 'single'
  addForm.value = ''
  addForm.rangeStart = ''
  addForm.rangeEnd = ''
  addForm.blockType = 'permanent'
  addForm.blockDays = 7
  addDialogVisible.value = true
}

function resetAddForm() {
  addForm.entry_type = 'single'
  addForm.value = ''
  addForm.rangeStart = ''
  addForm.rangeEnd = ''
  addForm.blockType = 'permanent'
  addForm.blockDays = 7
}

function validateIP(value) {
  const ipv4 = /^(\d{1,3}\.){3}\d{1,3}$/
  const ipv6 = /^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/
  return ipv4.test(value) || ipv6.test(value)
}

function validateCIDR(value) {
  const parts = value.split('/')
  if (parts.length !== 2) return false
  const ip = parts[0]
  const prefix = parseInt(parts[1], 10)
  if (!validateIP(ip)) return false
  const isV6 = ip.includes(':')
  const maxPrefix = isV6 ? 128 : 32
  return !isNaN(prefix) && prefix >= 0 && prefix <= maxPrefix
}

function confirmAddEntry() {
  const entryType = addListType.value === 'blacklist' ? 'single' : addForm.entry_type

  // For range type, assemble from start + end
  if (entryType === 'range') {
    const start = addForm.rangeStart.trim()
    const end = addForm.rangeEnd.trim()
    if (!start || !end) {
      ElMessage.warning(t('settings_access_control.required_value'))
      return
    }
    if (!validateIP(start) || !validateIP(end)) {
      ElMessage.warning(t('settings_access_control.invalid_range'))
      return
    }
    entries.value.push({ entry_type: 'range', value: `${start}~${end}`, block_days: null })
    addDialogVisible.value = false
    return
  }

  const value = addForm.value.trim()
  if (!value) {
    ElMessage.warning(t('settings_access_control.required_value'))
    return
  }

  if (entryType === 'single' && !validateIP(value)) {
    ElMessage.warning(t('settings_access_control.invalid_ip'))
    return
  }
  if (entryType === 'cidr' && !validateCIDR(value)) {
    ElMessage.warning(t('settings_access_control.invalid_cidr'))
    return
  }

  const entry = {
    entry_type: entryType,
    value: value,
    block_days: null,
  }

  if (addListType.value === 'blacklist') {
    entry.block_days = addForm.blockType === 'permanent' ? null : addForm.blockDays
    if (addForm.blockType === 'days' && (!addForm.blockDays || addForm.blockDays < 1)) {
      ElMessage.warning(t('settings_access_control.required_block_days'))
      return
    }
  }

  entries.value.push(entry)
  addDialogVisible.value = false
}

function removeEntry(index) {
  entries.value.splice(index, 1)
}

async function saveSettings() {
  saving.value = true
  try {
    const payload = {
      mode: mode.value,
      entries: entries.value.map(e => ({
        entry_type: e.entry_type,
        value: e.value,
        block_days: e.block_days,
      })),
    }
    await api.put('/settings/access-control', payload)
    ElMessage.success(t('settings_access_control.save_success'))
    originalMode.value = mode.value
    await loadSettings()
  } catch {}
  finally { saving.value = false }
}
</script>

<style scoped>
.acl-page {
  width: 100%;
}
.acl-card {
  max-width: 800px;
}
.list-card {
  margin-top: 16px;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.card-header > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Mode selection cards */
.mode-group {
  display: flex;
  width: 100%;
}
.mode-radio-group {
  display: flex;
  flex-direction: row;
  width: 100%;
  gap: 20px;
}
:deep(.mode-radio-group.el-radio-group) {
  display: flex;
  flex-direction: row;
  width: 100%;
}
.mode-card {
  display: flex;
  flex-direction: column;
  flex: 1;
  padding: 0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: var(--mode-card-bg);
  border: 1px solid var(--mode-card-border);
  box-sizing: border-box;
  overflow: hidden;
}
.mode-card:hover {
  border-color: #c0c4cc;
}
.mode-card.active {
  border-color: #409eff;
}
.mode-card-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: var(--mode-card-header-bg);
}
.mode-card.active .mode-card-header {
  background: var(--mode-card-active-header-bg);
}
.mode-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-left: 8px;
}
.mode-card-divider {
  height: 1px;
  background: var(--mode-card-border);
  width: 100%;
}
.mode-card.active .mode-card-divider {
  background: var(--mode-card-active-divider);
}
.mode-card-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px 16px;
  gap: 16px;
}
.mode-icon {
  width: 80px;
  height: 80px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}
.disabled-icon {
  background: #909399;
}
.whitelist-icon {
  background: #67c23a;
}
.blacklist-icon {
  background: #f56c6c;
}
.mode-hint {
  font-size: 13px;
  color: var(--el-text-color-regular);
  line-height: 1.5;
  text-align: center;
  height: 40px;
}
.mode-radio {
  margin-right: 0;
}

/* IP value styling */
.ip-value {
  background: var(--el-fill-color-light);
  padding: 2px 8px;
  border-radius: 4px;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
  color: var(--el-text-color-primary);
}

/* Save bar */
.save-bar {
  margin-top: 20px;
  max-width: 800px;
}
</style>
