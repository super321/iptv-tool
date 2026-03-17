<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <h3 style="margin: 0">{{ $t('epg_sources.title') }}</h3>
      <el-button type="primary" @click="showCreate">{{ $t('epg_sources.add') }}</el-button>
    </div>

    <el-table :data="sources" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" :label="$t('common.name')" min-width="140" show-overflow-tooltip />
      <el-table-column prop="description" :label="$t('common.description')" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>
      <el-table-column prop="type" :label="$t('common.type')" width="120">
        <template #default="{ row }">
          <el-tag :type="row.type === 'iptv' ? 'danger' : ''" size="small">{{ row.type === 'iptv' ? 'IPTV' : 'XMLTV' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="cron_time" :label="$t('epg_sources.scheduled_refresh')" width="140">
        <template #default="{ row }">{{ row.cron_time || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('epg_sources.channel_count')" width="120" align="center">
        <template #default="{ row }">{{ row.channel_count || 0 }}</template>
      </el-table-column>
      <el-table-column :label="$t('epg_sources.program_count')" width="120" align="center">
        <template #default="{ row }">{{ row.program_count || 0 }}</template>
      </el-table-column>
      <el-table-column prop="status" :label="$t('common.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'" size="small">{{ row.status ? $t('common.enabled') : $t('common.disabled') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.update_time')" width="200">
        <template #default="{ row }">
          <div v-if="row.is_syncing" style="display: flex; align-items: center; gap: 6px; color: #409eff">
            <el-icon class="is-loading" :size="16"><Loading /></el-icon>
            <span>{{ $t('common.syncing') }}</span>
          </div>
          <div v-else-if="row.last_fetched_at" style="display: flex; align-items: center; gap: 6px">
            <el-tooltip v-if="row.last_error" :content="row.last_error" placement="top" :show-after="300">
              <el-icon color="#f56c6c" :size="16" style="cursor: pointer; flex-shrink: 0"><CircleCloseFilled /></el-icon>
            </el-tooltip>
            <el-icon v-else color="#67c23a" :size="16" style="flex-shrink: 0"><SuccessFilled /></el-icon>
            <span>{{ new Date(row.last_fetched_at).toLocaleString() }}</span>
          </div>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.operations')" width="180" fixed="right" align="center">
        <template #default="{ row }">
          <el-tooltip :content="$t('epg_sources.programs_title')" placement="top" :show-after="500">
            <el-button :icon="Notebook" size="small" circle @click="showPrograms(row)" />
          </el-tooltip>
          <el-tooltip :content="$t('epg_sources.tooltip_sync')" placement="top" :show-after="500">
            <el-button :icon="Refresh" size="small" circle type="warning" @click="triggerFetch(row)" />
          </el-tooltip>
          <el-tooltip :content="$t('common.edit')" placement="top" :show-after="500">
            <el-button :icon="Edit" size="small" circle type="primary" @click="showEdit(row)" />
          </el-tooltip>
          <el-tooltip :content="$t('common.delete')" placement="top" :show-after="500">
            <el-button :icon="Delete" size="small" circle type="danger" @click="handleDelete(row)" />
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create/Edit Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('epg_sources.edit_title') : $t('epg_sources.add_title')" width="580px" destroy-on-close :close-on-click-modal="false">
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="110px">
        <el-form-item :label="$t('common.name')" prop="name">
          <el-input v-model.trim="form.name" />
        </el-form-item>
        <el-form-item :label="$t('common.description')" prop="description">
          <el-input v-model.trim="form.description" :placeholder="$t('common.optional_description')" />
        </el-form-item>
        <el-form-item :label="$t('common.type')" prop="type" v-if="!isEdit">
          <el-select v-model="form.type" style="width: 100%" @change="onTypeChange">
            <el-option :label="$t('epg_sources.type_xmltv')" value="network_xmltv" />
            <el-option :label="$t('epg_sources.type_iptv')" value="iptv" />
          </el-select>
        </el-form-item>

        <!-- XMLTV fields -->
        <el-form-item label="XMLTV URL" v-if="form.type === 'network_xmltv'" prop="url">
          <el-input v-model.trim="form.url" :placeholder="$t('epg_sources.xmltv_url_placeholder')" />
        </el-form-item>

        <!-- IPTV fields -->
        <template v-if="form.type === 'iptv'">
          <el-form-item :label="$t('epg_sources.linked_source')" prop="live_source_id" v-if="!isEdit">
            <el-select v-model="form.live_source_id" style="width: 100%" :placeholder="$t('epg_sources.select_unlinked')"
              :loading="unlinkedLoading" :no-data-text="$t('epg_sources.no_unlinked')">
              <el-option v-for="s in unlinkedSources" :key="s.id" :label="`${s.name} (ID: ${s.id})`" :value="s.id" />
            </el-select>
            <div style="color: #909399; font-size: 12px; margin-top: 4px">
              {{ $t('epg_sources.only_unlinked') }}
            </div>
          </el-form-item>
          <el-form-item :label="$t('epg_sources.linked_source')" v-if="isEdit">
            <el-input :model-value="linkedSourceName" disabled />
          </el-form-item>
          <el-form-item :label="$t('epg_sources.epg_strategy')" prop="epg_strategy">
            <el-select v-model="form.epg_strategy" style="width: 100%">
              <el-option v-for="opt in epgStrategies" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <div style="color: #909399; font-size: 12px; margin-top: 4px">
              {{ $t('epg_sources.epg_strategy_help') }}
            </div>
          </el-form-item>
        </template>

        <el-form-item :label="$t('epg_sources.scheduled_refresh')">
          <el-select v-model="form.cron_time" clearable :placeholder="$t('epg_sources.no_scheduled_refresh')" style="width: 100%">
            <el-option v-for="opt in intervalOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('common.status')" v-if="isEdit">
          <el-switch v-model="form.status" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- Programs Drill-down Dialog -->
    <el-dialog v-model="programsVisible" :title="$t('epg_sources.programs_title')" width="750px" destroy-on-close :close-on-click-modal="false">
      <!-- Breadcrumb navigation -->
      <div style="margin-bottom: 16px; display: flex; align-items: center; gap: 4px; color: #606266; font-size: 14px">
        <el-link :underline="false" @click="drillLevel = 1" :type="drillLevel === 1 ? 'primary' : 'default'">
          {{ $t('epg_sources.channel_list') }}
        </el-link>
        <template v-if="drillLevel >= 2">
          <span style="color: #c0c4cc">/</span>
          <el-link :underline="false" @click="drillLevel = 2" :type="drillLevel === 2 ? 'primary' : 'default'">
            {{ drillChannel }}
          </el-link>
        </template>
        <template v-if="drillLevel === 3">
          <span style="color: #c0c4cc">/</span>
          <el-link :underline="false" type="primary">{{ drillDate }}</el-link>
        </template>
      </div>

      <!-- Level 1: Channel list -->
      <div v-if="drillLevel === 1">
        <div style="margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center">
          <p style="margin: 0; color: #909399; font-size: 13px">
            {{ $t('epg_sources.channels_total', { count: filteredEpgChannels.length }) }} {{ drillSearch ? $t('epg_sources.channels_filtered') : '' }}
          </p>
          <el-input v-model="drillSearch" :placeholder="$t('epg_sources.search_channel')" style="width: 200px" size="small" clearable @input="handleSearchChange" />
        </div>
        <el-table :data="paginatedEpgChannels" v-loading="drillLoading" max-height="400" border stripe size="small">
          <el-table-column prop="channel" :label="$t('epg_sources.col_channel_id')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="channel_name" :label="$t('epg_sources.col_channel_name')" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ row.channel_name || row.channel }}</template>
          </el-table-column>
          <el-table-column prop="count" :label="$t('epg_sources.program_count')" width="140" />
          <el-table-column :label="$t('common.operations')" width="120" align="center">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="drillToDate(row.channel)">{{ $t('epg_sources.col_view') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div style="margin-top: 12px; display: flex; justify-content: flex-end">
          <el-pagination
            v-model:current-page="drillCurrentPage"
            v-model:page-size="drillPageSize"
            :page-sizes="[50, 100, 200, 500]"
            layout="total, sizes, prev, pager, next"
            :total="filteredEpgChannels.length"
            size="small"
          />
        </div>
      </div>

      <!-- Level 2: Date list -->
      <el-table v-if="drillLevel === 2" :data="epgDates" v-loading="drillLoading" max-height="400" border stripe size="small">
        <el-table-column prop="date" :label="$t('epg_sources.col_date')" min-width="200" />
        <el-table-column prop="count" :label="$t('epg_sources.program_count')" width="140" />
        <el-table-column :label="$t('common.operations')" width="120" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="drillToPrograms(row.date)">{{ $t('epg_sources.col_view') }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Level 3: Program list -->
      <el-table v-if="drillLevel === 3" :data="epgPrograms" v-loading="drillLoading" max-height="400" border stripe size="small">
        <el-table-column :label="$t('epg_sources.col_time')" width="160">
          <template #default="{ row }">
            {{ formatTime(row.start_time) }} - {{ formatTime(row.end_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="title" :label="$t('epg_sources.col_program_name')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="desc" :label="$t('common.description')" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ row.desc || '-' }}</template>
        </el-table-column>
      </el-table>

    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Notebook, Refresh, Edit, Delete, SuccessFilled, CircleCloseFilled, Loading } from '@element-plus/icons-vue'
import api from '../api'

const { t, locale } = useI18n()

const sources = ref([])
const loading = ref(false)
let pollingTimer = null

onUnmounted(() => {
  if (pollingTimer) clearInterval(pollingTimer)
})

const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const submitting = ref(false)
const formRef = ref()
const intervalOptions = ref([])
const epgStrategies = ref([])
const unlinkedSources = ref([])
const unlinkedLoading = ref(false)

// Programs drill-down state
const programsVisible = ref(false)
const programsSourceId = ref(null)
const drillLevel = ref(1)
const drillChannel = ref('')
const drillDate = ref('')
const drillLoading = ref(false)
const epgChannels = ref([])
const epgDates = ref([])
const epgPrograms = ref([])

// Added pagination & search state
const drillSearch = ref('')
const drillCurrentPage = ref(1)
const drillPageSize = ref(50)

const filteredEpgChannels = computed(() => {
  let result = epgChannels.value
  if (drillSearch.value) {
    const q = drillSearch.value.toLowerCase()
    result = result.filter(c => 
      (c.channel && c.channel.toLowerCase().includes(q)) || 
      (c.channel_name && c.channel_name.toLowerCase().includes(q))
    )
  }
  return result
})

const paginatedEpgChannels = computed(() => {
  const start = (drillCurrentPage.value - 1) * drillPageSize.value
  const end = start + drillPageSize.value
  return filteredEpgChannels.value.slice(start, end)
})

function handleSearchChange() {
  drillCurrentPage.value = 1
}

const defaultForm = () => ({
  name: '', description: '', type: 'network_xmltv', url: '', cron_time: '',
  live_source_id: null, epg_strategy: 'auto', status: true,
})
const form = reactive(defaultForm())

const formRules = computed(() => ({
  name: [{ required: true, message: t('common.required_name'), trigger: 'blur' }],
  type: [{ required: true, message: t('common.required_type'), trigger: 'change' }],
  url: [{ required: true, message: t('epg_sources.required_url'), trigger: 'blur' }],
  live_source_id: [{ required: true, message: t('epg_sources.required_live_source'), trigger: 'change' }],
}))

// Computed: linked source display for edit mode
const linkedSourceName = computed(() => {
  if (!isEdit.value) return ''
  const editSource = sources.value.find(s => s.id === editId.value)
  if (!editSource || !editSource.live_source_id) return t('epg_sources.no_linked')
  return t('epg_sources.linked_source_id', { id: editSource.live_source_id })
})

onMounted(async () => {
  await loadSources()
  try {
    const [intervalRes, epgRes] = await Promise.all([
      api.get('/settings/interval-options'),
      api.get('/settings/epg-strategies'),
    ])
    intervalOptions.value = intervalRes.data
    epgStrategies.value = epgRes.data
  } catch {}
})

async function loadSources(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    const { data } = await api.get('/epg-sources')
    sources.value = data || []

    // Check polling
    const hasSyncing = sources.value.some(s => s.is_syncing)
    if (hasSyncing && !pollingTimer) {
      pollingTimer = setInterval(() => loadSources(false), 3000)
    } else if (!hasSyncing && pollingTimer) {
      clearInterval(pollingTimer)
      pollingTimer = null
    }
  } finally { if (showLoading) loading.value = false }
}

async function loadUnlinkedSources() {
  unlinkedLoading.value = true
  try {
    const { data } = await api.get('/live-sources/unlinked-iptv')
    unlinkedSources.value = data
  } catch {
    unlinkedSources.value = []
  } finally { unlinkedLoading.value = false }
}

function onTypeChange() {
  if (form.type === 'iptv') {
    loadUnlinkedSources()
  }
}

function getEpgStrategy(row) {
  if (row.type !== 'iptv' || !row.iptv_config) return 'auto'
  try {
    const cfg = JSON.parse(row.iptv_config)
    return cfg.epgStrategy || cfg.channelProgramAPI || 'auto'
  } catch { return 'auto' }
}

function showCreate() {
  isEdit.value = false
  editId.value = null
  Object.assign(form, defaultForm())
  dialogVisible.value = true
  // Pre-load unlinked sources if type is iptv
  if (form.type === 'iptv') {
    loadUnlinkedSources()
  }
}

function showEdit(row) {
  isEdit.value = true
  editId.value = row.id
  Object.assign(form, {
    name: row.name,
    description: row.description || '',
    type: row.type,
    url: row.url || '',
    cron_time: row.cron_time || '',
    live_source_id: row.live_source_id,
    epg_strategy: getEpgStrategy(row),
    status: row.status,
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value.validate()
  submitting.value = true
  try {
    if (isEdit.value) {
      const body = {
        name: form.name,
        description: form.description,
        url: form.url,
        cron_time: form.cron_time || '',
        status: form.status,
      }
      if (form.type === 'iptv') {
        body.epg_strategy = form.epg_strategy
      }
      await api.put(`/epg-sources/${editId.value}`, body)
      ElMessage.success(t('common.update_success'))
    } else {
      const body = {
        name: form.name,
        description: form.description,
        type: form.type,
        url: form.url,
        cron_time: form.cron_time || '',
      }
      if (form.type === 'iptv') {
        body.live_source_id = form.live_source_id
        body.epg_strategy = form.epg_strategy
      }
      await api.post('/epg-sources', body)
      ElMessage.success(t('common.create_success'))
    }
    dialogVisible.value = false
    await loadSources()
  } catch {}
  finally { submitting.value = false }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(
    t('epg_sources.delete_confirm', { name: row.name }),
    t('common.confirm_delete'),
    { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') }
  )
  await api.delete(`/epg-sources/${row.id}`)
  ElMessage.success(t('common.delete_success'))
  await loadSources()
}

async function triggerFetch(row) {
  await api.post(`/epg-sources/${row.id}/trigger`)
  ElMessage.success(t('common.trigger_success'))
  await loadSources(false)
}

// --- Programs drill-down ---
async function showPrograms(row) {
  programsSourceId.value = row.id
  drillLevel.value = 1
  drillChannel.value = ''
  drillDate.value = ''
  drillSearch.value = ''
  drillCurrentPage.value = 1
  programsVisible.value = true
  await loadEpgChannels()
}

async function loadEpgChannels() {
  drillLoading.value = true
  try {
    const { data } = await api.get(`/epg-sources/${programsSourceId.value}/channels`)
    epgChannels.value = data.channels || []
  } catch {
    epgChannels.value = []
  } finally { drillLoading.value = false }
}

async function drillToDate(channel) {
  drillChannel.value = channel
  drillLevel.value = 2
  drillLoading.value = true
  try {
    const { data } = await api.get(`/epg-sources/${programsSourceId.value}/dates`, {
      params: { channel }
    })
    epgDates.value = data.dates || []
  } catch {
    epgDates.value = []
  } finally { drillLoading.value = false }
}

async function drillToPrograms(date) {
  drillDate.value = date
  drillLevel.value = 3
  drillLoading.value = true
  try {
    const { data } = await api.get(`/epg-sources/${programsSourceId.value}/programs`, {
      params: { channel: drillChannel.value, date }
    })
    epgPrograms.value = data.programs || []
  } catch {
    epgPrograms.value = []
  } finally { drillLoading.value = false }
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  return d.toLocaleTimeString(locale.value === 'zh' ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit' })
}
</script>