<template>
  <div>
    <h3 style="margin: 0 0 16px">{{ $t('config_transfer.title') }}</h3>

    <el-tabs v-model="activeTab" type="border-card" class="config-tabs">
      <!-- ==================== EXPORT TAB ==================== -->
      <el-tab-pane :label="$t('config_transfer.tab_export')" name="export">
        <div class="section-header">
          <el-icon :size="20"><Upload /></el-icon>
          <div>
            <div class="section-title">{{ $t('config_transfer.export_title') }}</div>
            <div class="section-desc">{{ $t('config_transfer.export_desc') }}</div>
          </div>
        </div>

        <div class="module-header">
          <span class="module-label">{{ $t('config_transfer.select_modules') }}</span>
          <el-checkbox
            v-model="exportSelectAll"
            :indeterminate="exportIndeterminate"
            @change="onExportSelectAll"
          >{{ $t('config_transfer.select_all') }}</el-checkbox>
        </div>

        <div class="module-grid">
          <div
            v-for="mod in moduleOptions"
            :key="mod.value"
            class="module-card"
            :class="{ selected: exportModules.includes(mod.value), disabled: mod.forceDisabled }"
            @click="toggleExportModule(mod.value)"
          >
            <el-checkbox
              :model-value="exportModules.includes(mod.value)"
              :disabled="mod.forceDisabled"
              @click.stop
              @change="(val) => onModuleCheckChange(mod.value, val)"
            />
            <el-icon :size="28" class="module-icon"><component :is="mod.icon" /></el-icon>
            <div class="module-info">
              <div class="module-name">{{ $t(mod.label) }}</div>
              <div class="module-desc">{{ $t(mod.desc) }}</div>
            </div>
          </div>
        </div>

        <div v-if="publishDepWarning" class="dep-warning">
          <el-icon><Warning /></el-icon>
          <span>{{ $t('config_transfer.publish_dep_warning') }}</span>
        </div>

        <div style="margin-top: 24px; text-align: center">
          <el-button type="primary" size="large" :loading="exporting" :disabled="exportModules.length === 0" @click="doExport">
            <el-icon style="margin-right: 6px"><Download /></el-icon>
            {{ exporting ? $t('config_transfer.exporting') : $t('config_transfer.export_btn') }}
          </el-button>
        </div>
      </el-tab-pane>

      <!-- ==================== IMPORT TAB ==================== -->
      <el-tab-pane :label="$t('config_transfer.tab_import')" name="import">
        <div class="section-header">
          <el-icon :size="20"><Download /></el-icon>
          <div>
            <div class="section-title">{{ $t('config_transfer.import_title') }}</div>
            <div class="section-desc">{{ $t('config_transfer.import_desc') }}</div>
          </div>
        </div>

        <!-- Step 1: Upload -->
        <div v-if="importStep === 'upload'" class="import-upload-area">
          <el-upload
            drag
            :auto-upload="false"
            :show-file-list="false"
            accept=".zip"
            :on-change="onFileSelect"
          >
            <el-icon class="el-icon--upload" :size="48"><Upload /></el-icon>
            <div class="el-upload__text">{{ $t('config_transfer.upload_zip') }}</div>
            <template #tip>
              <div class="el-upload__tip">{{ $t('config_transfer.upload_hint') }}</div>
            </template>
          </el-upload>
          <div v-if="selectedFile" class="selected-file">
            <el-tag closable @close="selectedFile = null">{{ selectedFile.name }}</el-tag>
            <el-button type="primary" :loading="parsing" @click="parseZip" style="margin-left: 12px">
              {{ parsing ? $t('config_transfer.parsing') : $t('config_transfer.upload_btn') }}
            </el-button>
          </div>
        </div>

        <!-- Step 2: Preview & Confirm -->
        <div v-if="importStep === 'preview'">
          <!-- Version warning -->
          <el-alert
            v-if="parsedData.version_warning"
            :title="$t('config_transfer.version_warning')"
            :description="parsedData.version_warning"
            type="warning"
            show-icon
            :closable="false"
            style="margin-bottom: 16px"
          />

          <el-descriptions :column="2" border size="small" style="margin-bottom: 16px">
            <el-descriptions-item :label="$t('config_transfer.export_version')">{{ parsedData.manifest?.version }}</el-descriptions-item>
            <el-descriptions-item :label="$t('config_transfer.export_time')">{{ formatTime(parsedData.manifest?.exported_at) }}</el-descriptions-item>
          </el-descriptions>

          <h4 style="margin: 0 0 12px">{{ $t('config_transfer.import_preview') }}</h4>
          <el-table :data="parsedData.summaries" border stripe style="width: 100%">
            <el-table-column :label="$t('config_transfer.col_module')" prop="module" min-width="180">
              <template #default="{ row }">{{ getModuleLabel(row.module) }}</template>
            </el-table-column>
            <el-table-column :label="$t('config_transfer.col_count')" prop="count" width="120" align="center" />
          </el-table>

          <div style="margin-top: 20px; display: flex; gap: 12px; justify-content: center">
            <el-button @click="resetImport">{{ $t('config_transfer.re_upload') }}</el-button>
            <el-button type="primary" :loading="importing" @click="doImport">
              {{ importing ? $t('config_transfer.importing') : $t('config_transfer.confirm_import') }}
            </el-button>
          </div>
        </div>

        <!-- Step 3: Result -->
        <div v-if="importStep === 'result'">
          <el-alert
            :title="$t('config_transfer.import_complete')"
            type="success"
            show-icon
            :closable="false"
            style="margin-bottom: 16px"
          />

          <el-table :data="importResult" border stripe style="width: 100%">
            <el-table-column :label="$t('config_transfer.col_module')" prop="module" min-width="140">
              <template #default="{ row }">{{ getModuleLabel(row.module) }}</template>
            </el-table-column>
            <el-table-column :label="$t('config_transfer.result_total')" prop="total" width="80" align="center" />
            <el-table-column :label="$t('config_transfer.result_success')" width="80" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.success > 0" type="success" size="small">{{ row.success }}</el-tag>
                <span v-else>0</span>
              </template>
            </el-table-column>
            <el-table-column :label="$t('config_transfer.result_failed')" width="80" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.failed > 0" type="danger" size="small">{{ row.failed }}</el-tag>
                <span v-else>0</span>
              </template>
            </el-table-column>
            <el-table-column :label="$t('config_transfer.result_skipped')" width="80" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.skipped > 0" type="warning" size="small">{{ row.skipped }}</el-tag>
                <span v-else>0</span>
              </template>
            </el-table-column>
            <el-table-column :label="$t('config_transfer.result_details')" min-width="200">
              <template #default="{ row }">
                <div v-if="row.details && row.details.length > 0">
                  <el-popover
                    placement="left"
                    trigger="click"
                    :width="400"
                  >
                    <template #reference>
                      <el-button link type="primary" size="small">
                        {{ $t('config_transfer.result_details') }} ({{ row.details.length }})
                      </el-button>
                    </template>
                    <div class="detail-list">
                      <div v-for="(d, idx) in row.details" :key="idx" class="detail-item">{{ d }}</div>
                    </div>
                  </el-popover>
                </div>
                <span v-else style="color: var(--el-text-color-placeholder)">—</span>
              </template>
            </el-table-column>
          </el-table>

          <div style="margin-top: 20px; text-align: center">
            <el-button @click="resetImport">{{ $t('config_transfer.re_upload') }}</el-button>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Upload, Download, VideoCamera, Picture, Guide, Share, Stopwatch, Lock, Warning } from '@element-plus/icons-vue'
import api from '../api'

const { t } = useI18n()

// ---- Module definitions ----
const moduleOptions = [
  { value: 'sources', label: 'config_transfer.module_sources', desc: 'config_transfer.module_sources_desc', icon: VideoCamera },
  { value: 'logos', label: 'config_transfer.module_logos', desc: 'config_transfer.module_logos_desc', icon: Picture },
  { value: 'rules', label: 'config_transfer.module_rules', desc: 'config_transfer.module_rules_desc', icon: Guide },
  { value: 'publish', label: 'config_transfer.module_publish', desc: 'config_transfer.module_publish_desc', icon: Share },
  { value: 'detect', label: 'config_transfer.module_detect', desc: 'config_transfer.module_detect_desc', icon: Stopwatch },
  { value: 'access_control', label: 'config_transfer.module_access_control', desc: 'config_transfer.module_access_control_desc', icon: Lock },
]

// ---- State ----
const activeTab = ref('export')

// Export state
const exportModules = ref(['sources', 'logos', 'rules', 'publish', 'detect', 'access_control'])
const exporting = ref(false)
const publishDepWarning = ref(false)

// Import state
const importStep = ref('upload') // upload -> preview -> result
const selectedFile = ref(null)
const parsing = ref(false)
const importing = ref(false)
const parsedData = ref({})
const importResult = ref([])

// ---- Export logic ----
const exportSelectAll = computed({
  get: () => exportModules.value.length === moduleOptions.length,
  set: () => {}
})
const exportIndeterminate = computed(() => {
  return exportModules.value.length > 0 && exportModules.value.length < moduleOptions.length
})

function onExportSelectAll(val) {
  if (val) {
    exportModules.value = moduleOptions.map(m => m.value)
  } else {
    exportModules.value = []
  }
}

function toggleExportModule(value) {
  const idx = exportModules.value.indexOf(value)
  if (idx >= 0) {
    onModuleCheckChange(value, false)
  } else {
    onModuleCheckChange(value, true)
  }
}

function onModuleCheckChange(value, checked) {
  const mods = [...exportModules.value]
  if (checked) {
    if (!mods.includes(value)) mods.push(value)
    // If publish is checked, force-check sources and rules
    if (value === 'publish') {
      if (!mods.includes('sources')) mods.push('sources')
      if (!mods.includes('rules')) mods.push('rules')
    }
  } else {
    const idx = mods.indexOf(value)
    if (idx >= 0) mods.splice(idx, 1)
    // If sources or rules unchecked, force-uncheck publish
    if (value === 'sources' || value === 'rules') {
      const pubIdx = mods.indexOf('publish')
      if (pubIdx >= 0) mods.splice(pubIdx, 1)
    }
  }
  exportModules.value = mods
}

// Show dependency warning when publish is checked
watch(exportModules, (mods) => {
  publishDepWarning.value = mods.includes('publish')
}, { immediate: true })

async function doExport() {
  if (exportModules.value.length === 0) {
    ElMessage.warning(t('config_transfer.no_modules_selected'))
    return
  }
  exporting.value = true
  try {
    const response = await api.post('/config/export', { modules: exportModules.value }, { responseType: 'blob' })

    // Extract filename from Content-Disposition header
    const disposition = response.headers['content-disposition']
    let filename = 'iptv-config.zip'
    if (disposition) {
      const match = disposition.match(/filename="?(.+?)"?$/i)
      if (match) filename = match[1]
    }

    // Trigger download
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', filename)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)

    ElMessage.success(t('config_transfer.export_success'))
  } catch (err) {
    // With responseType: 'blob', error responses are also Blob objects
    // Parse the blob to extract the actual error message
    if (err?.response?.data instanceof Blob) {
      try {
        const text = await err.response.data.text()
        const json = JSON.parse(text)
        if (json.error) ElMessage.error(json.error)
      } catch {
        // Fallback if blob parsing fails
      }
    }
  } finally {
    exporting.value = false
  }
}

// ---- Import logic ----
function onFileSelect(file) {
  selectedFile.value = file.raw
}

async function parseZip() {
  if (!selectedFile.value) return
  parsing.value = true
  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    const { data } = await api.post('/config/import/parse', formData)
    parsedData.value = data
    if (!data.summaries || data.summaries.length === 0) {
      ElMessage.warning(t('config_transfer.no_data'))
      return
    }
    importStep.value = 'preview'
    ElMessage.success(t('config_transfer.parse_success'))
  } catch {
    // Error handled by interceptor
  } finally {
    parsing.value = false
  }
}

async function doImport() {
  if (!selectedFile.value) return
  importing.value = true
  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    const { data } = await api.post('/config/import/execute', formData)
    importResult.value = data.modules || []
    importStep.value = 'result'
    ElMessage.success(t('config_transfer.import_complete'))
  } catch {
    // Error handled by interceptor
  } finally {
    importing.value = false
  }
}

function resetImport() {
  importStep.value = 'upload'
  selectedFile.value = null
  parsedData.value = {}
  importResult.value = []
}

// ---- Helpers ----
function getModuleLabel(moduleKey) {
  const mod = moduleOptions.find(m => m.value === moduleKey)
  return mod ? t(mod.label) : moduleKey
}

function formatTime(isoStr) {
  if (!isoStr) return '—'
  try {
    return new Date(isoStr).toLocaleString()
  } catch {
    return isoStr
  }
}
</script>

<style scoped>
.config-tabs {
  max-width: 900px;
}
.config-tabs :deep(.el-tabs__content) {
  padding: 24px;
}
.section-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 24px;
  padding: 16px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}
.section-header .el-icon {
  color: var(--el-color-primary);
  margin-top: 2px;
}
.section-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
}
.section-desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.module-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.module-label {
  font-weight: 500;
  font-size: 14px;
}
.module-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.module-card {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: var(--el-bg-color);
}
.module-card:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
}
.module-card.selected {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.module-card.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.module-icon {
  color: var(--el-color-primary);
  flex-shrink: 0;
  margin-top: 2px;
}
.module-info {
  flex: 1;
  min-width: 0;
}
.module-name {
  font-weight: 500;
  font-size: 14px;
  margin-bottom: 4px;
}
.module-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}
.dep-warning {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding: 8px 12px;
  border-radius: 6px;
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
  font-size: 13px;
}
.import-upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}
.import-upload-area :deep(.el-upload-dragger) {
  width: 100%;
  padding: 40px 20px;
}
.selected-file {
  display: flex;
  align-items: center;
  margin-top: 4px;
}
.detail-list {
  max-height: 300px;
  overflow-y: auto;
}
.detail-item {
  padding: 4px 0;
  font-size: 13px;
  color: var(--el-text-color-regular);
  border-bottom: 1px dashed var(--el-border-color-extra-light);
}
.detail-item:last-child {
  border-bottom: none;
}
</style>
