<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <h3 style="margin: 0">EPG源管理</h3>
      <el-button type="primary" @click="showCreate">新增EPG源</el-button>
    </div>

    <el-table :data="sources" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" width="120" show-overflow-tooltip />
      <el-table-column prop="description" label="描述" min-width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="row.type === 'iptv' ? 'danger' : ''" size="small">{{ row.type === 'iptv' ? 'IPTV' : 'XMLTV' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="cron_time" label="定时刷新" width="100">
        <template #default="{ row }">{{ row.cron_time || '-' }}</template>
      </el-table-column>
      <el-table-column label="频道数" width="80" align="center">
        <template #default="{ row }">{{ row.channel_count || 0 }}</template>
      </el-table-column>
      <el-table-column label="节目数" width="80" align="center">
        <template #default="{ row }">{{ row.program_count || 0 }}</template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'" size="small">{{ row.status ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="更新时间" width="200">
        <template #default="{ row }">
          <div v-if="row.last_fetched_at" style="display: flex; align-items: center; gap: 6px">
            <el-tooltip v-if="row.last_error" :content="row.last_error" placement="top" :show-after="300">
              <el-icon color="#f56c6c" :size="16" style="cursor: pointer; flex-shrink: 0"><CircleCloseFilled /></el-icon>
            </el-tooltip>
            <el-icon v-else color="#67c23a" :size="16" style="flex-shrink: 0"><SuccessFilled /></el-icon>
            <span>{{ new Date(row.last_fetched_at).toLocaleString() }}</span>
          </div>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right" align="center">
        <template #default="{ row }">
          <el-tooltip content="节目单" placement="top" :show-after="500">
            <el-button :icon="Notebook" size="small" circle @click="showPrograms(row)" />
          </el-tooltip>
          <el-tooltip content="刷新抓取" placement="top" :show-after="500">
            <el-button :icon="Refresh" size="small" circle type="warning" @click="triggerFetch(row)" />
          </el-tooltip>
          <el-tooltip content="编辑" placement="top" :show-after="500">
            <el-button :icon="Edit" size="small" circle type="primary" @click="showEdit(row)" />
          </el-tooltip>
          <el-tooltip content="删除" placement="top" :show-after="500">
            <el-button :icon="Delete" size="small" circle type="danger" @click="handleDelete(row)" />
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create/Edit Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑EPG源' : '新增EPG源'" width="580px" destroy-on-close :close-on-click-modal="false">
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="110px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" placeholder="可选的描述信息" />
        </el-form-item>
        <el-form-item label="类型" prop="type" v-if="!isEdit">
          <el-select v-model="form.type" style="width: 100%" @change="onTypeChange">
            <el-option label="网络XMLTV" value="network_xmltv" />
            <el-option label="IPTV (STB模拟)" value="iptv" />
          </el-select>
        </el-form-item>

        <!-- XMLTV fields -->
        <el-form-item label="XMLTV URL" v-if="form.type === 'network_xmltv'" prop="url">
          <el-input v-model="form.url" placeholder="http://example.com/epg.xml 或 .xml.gz" />
        </el-form-item>

        <!-- IPTV fields -->
        <template v-if="form.type === 'iptv'">
          <el-form-item label="关联直播源" prop="live_source_id" v-if="!isEdit">
            <el-select v-model="form.live_source_id" style="width: 100%" placeholder="选择一个未关联的IPTV直播源"
              :loading="unlinkedLoading" no-data-text="没有可用的未关联IPTV直播源">
              <el-option v-for="s in unlinkedSources" :key="s.id" :label="`${s.name} (ID: ${s.id})`" :value="s.id" />
            </el-select>
            <div style="color: #909399; font-size: 12px; margin-top: 4px">
              仅显示尚未关联EPG源的IPTV直播源
            </div>
          </el-form-item>
          <el-form-item label="关联直播源" v-if="isEdit">
            <el-input :model-value="linkedSourceName" disabled />
          </el-form-item>
          <el-form-item label="EPG策略" prop="epg_strategy">
            <el-select v-model="form.epg_strategy" style="width: 100%">
              <el-option v-for="opt in epgStrategies" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <div style="color: #909399; font-size: 12px; margin-top: 4px">
              选择"自动检测"将依次尝试所有策略，成功后自动记录
            </div>
          </el-form-item>
        </template>

        <el-form-item label="定时刷新">
          <el-select v-model="form.cron_time" clearable placeholder="不定时刷新" style="width: 100%">
            <el-option v-for="opt in cronOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态" v-if="isEdit">
          <el-switch v-model="form.status" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- Programs Drill-down Dialog -->
    <el-dialog v-model="programsVisible" title="节目单" width="750px" destroy-on-close :close-on-click-modal="false">
      <!-- Breadcrumb navigation -->
      <div style="margin-bottom: 16px; display: flex; align-items: center; gap: 4px; color: #606266; font-size: 14px">
        <el-link :underline="false" @click="drillLevel = 1" :type="drillLevel === 1 ? 'primary' : 'default'">
          频道列表
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
      <el-table v-if="drillLevel === 1" :data="epgChannels" v-loading="drillLoading" max-height="400" border stripe size="small">
        <el-table-column prop="channel" label="频道ID" width="180" show-overflow-tooltip />
        <el-table-column prop="channel_name" label="频道名称" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ row.channel_name || row.channel }}</template>
        </el-table-column>
        <el-table-column prop="count" label="节目数" width="100" />
        <el-table-column label="操作" width="100" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="drillToDate(row.channel)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Level 2: Date list -->
      <el-table v-if="drillLevel === 2" :data="epgDates" v-loading="drillLoading" max-height="400" border stripe size="small">
        <el-table-column prop="date" label="日期" min-width="200" />
        <el-table-column prop="count" label="节目数" width="100" />
        <el-table-column label="操作" width="100" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="drillToPrograms(row.date)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Level 3: Program list -->
      <el-table v-if="drillLevel === 3" :data="epgPrograms" v-loading="drillLoading" max-height="400" border stripe size="small">
        <el-table-column label="时间" width="160">
          <template #default="{ row }">
            {{ formatTime(row.start_time) }} - {{ formatTime(row.end_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="title" label="节目名称" min-width="200" show-overflow-tooltip />
        <el-table-column prop="desc" label="描述" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ row.desc || '-' }}</template>
        </el-table-column>
      </el-table>

      <p v-if="drillLevel === 1" style="margin: 12px 0 0; color: #909399; font-size: 13px">
        共 {{ epgChannels.length }} 个频道
      </p>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Notebook, Refresh, Edit, Delete, SuccessFilled, CircleCloseFilled } from '@element-plus/icons-vue'
import api from '../api'

const sources = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const submitting = ref(false)
const formRef = ref()
const cronOptions = ref([])
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

const defaultForm = () => ({
  name: '', description: '', type: 'network_xmltv', url: '', cron_time: '',
  live_source_id: null, epg_strategy: 'auto', status: true,
})
const form = reactive(defaultForm())

const formRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  url: [{ required: true, message: '请输入XMLTV URL', trigger: 'blur' }],
  live_source_id: [{ required: true, message: '请选择关联的IPTV直播源', trigger: 'change' }],
}

// Computed: linked source display for edit mode
const linkedSourceName = computed(() => {
  if (!isEdit.value) return ''
  const editSource = sources.value.find(s => s.id === editId.value)
  if (!editSource || !editSource.live_source_id) return '无关联'
  return `直播源 ID: ${editSource.live_source_id}`
})

onMounted(async () => {
  await loadSources()
  try {
    const [cronRes, epgRes] = await Promise.all([
      api.get('/settings/cron-options'),
      api.get('/settings/epg-strategies'),
    ])
    cronOptions.value = cronRes.data
    epgStrategies.value = epgRes.data
  } catch {}
})

async function loadSources() {
  loading.value = true
  try {
    const { data } = await api.get('/epg-sources')
    sources.value = data
  } finally { loading.value = false }
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
        cron_time: form.cron_time,
        status: form.status,
      }
      if (form.type === 'iptv') {
        body.epg_strategy = form.epg_strategy
      }
      await api.put(`/epg-sources/${editId.value}`, body)
      ElMessage.success('更新成功')
    } else {
      const body = {
        name: form.name,
        description: form.description,
        type: form.type,
        url: form.url,
        cron_time: form.cron_time,
      }
      if (form.type === 'iptv') {
        body.live_source_id = form.live_source_id
        body.epg_strategy = form.epg_strategy
      }
      await api.post('/epg-sources', body)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await loadSources()
  } catch {}
  finally { submitting.value = false }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(
    `确定删除EPG源 "${row.name}"？关联的节目数据将被清除。`,
    '确认删除',
    { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' }
  )
  await api.delete(`/epg-sources/${row.id}`)
  ElMessage.success('删除成功')
  await loadSources()
}

async function triggerFetch(row) {
  await api.post(`/epg-sources/${row.id}/trigger`)
  ElMessage.success('已触发刷新')
}

// --- Programs drill-down ---
async function showPrograms(row) {
  programsSourceId.value = row.id
  drillLevel.value = 1
  drillChannel.value = ''
  drillDate.value = ''
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
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}
</script>
