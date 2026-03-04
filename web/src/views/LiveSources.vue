<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <h3 style="margin: 0">直播源管理</h3>
      <el-button type="primary" @click="showCreate">新增直播源</el-button>
    </div>

    <el-table :data="sources" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" width="120" show-overflow-tooltip />
      <el-table-column prop="description" label="描述" min-width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="120">
        <template #default="{ row }">
          <el-tag :type="typeTagMap[row.type]" size="small">{{ typeNameMap[row.type] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="cron_time" label="定时刷新" width="100">
        <template #default="{ row }">{{ row.cron_time || '-' }}</template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'" size="small">{{ row.status ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="channel_count" label="频道数" width="80" align="center">
        <template #default="{ row }">
          <el-tag type="info" size="small">{{ row.channel_count || 0 }}</el-tag>
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
          <el-tooltip content="频道列表" placement="top" :show-after="500">
            <el-button :icon="List" size="small" circle @click="showChannels(row)" />
          </el-tooltip>
          <el-tooltip content="同步" placement="top" :show-after="500">
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
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑直播源' : '新增直播源'" width="680px" destroy-on-close :close-on-click-modal="false">
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="120px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" placeholder="可选的描述信息" />
        </el-form-item>
        <el-form-item label="类型" prop="type" v-if="!isEdit">
          <el-select v-model="form.type" style="width: 100%" @change="onTypeChange">
            <el-option label="IPTV (STB模拟)" value="iptv" />
            <el-option label="网络订阅URL" value="network_url" />
            <el-option label="手动输入" value="network_manual" />
          </el-select>
        </el-form-item>

        <!-- network_url fields -->
        <el-form-item label="订阅URL" v-if="form.type === 'network_url'" prop="url">
          <el-input v-model="form.url" placeholder="http://example.com/live.m3u 或 live.txt (系统自动识别M3U或TXT格式)" />
        </el-form-item>

        <!-- network_manual fields -->
        <el-form-item label="内容" v-if="form.type === 'network_manual'" prop="content">
          <el-input v-model="form.content" type="textarea" :rows="8" placeholder="支持纯文本粘贴，系统会自动识别格式：&#10;[M3U格式]&#10;#EXTM3U&#10;#EXTINF:-1,CCTV1&#10;http://...&#10;&#10;[TXT格式]&#10;央视,#genre#&#10;CCTV1,http://..." />
        </el-form-item>

        <!-- === IPTV specific fields === -->
        <template v-if="form.type === 'iptv'">
          <el-divider content-position="left">IPTV 平台配置</el-divider>

          <el-form-item label="IPTV平台" prop="iptv.platform">
            <el-select v-model="form.iptv.platform" style="width: 100%">
              <el-option label="华为 (Huawei)" value="huawei" />
            </el-select>
          </el-form-item>

          <el-form-item label="服务器地址" prop="iptv.serverHost">
            <el-input v-model="form.iptv.serverHost" placeholder="例如: 182.138.3.142:8082" />
          </el-form-item>

          <el-form-item label="运营商" prop="iptv.providerSuffix">
            <el-select v-model="form.iptv.providerSuffix" style="width: 100%">
              <el-option label="CTC (电信)" value="CTC" />
              <el-option label="CU (联通)" value="CU" />
            </el-select>
          </el-form-item>

          <el-form-item label="加密密钥" prop="iptv.key">
            <div style="display: flex; gap: 8px; width: 100%">
              <el-input v-model="form.iptv.key" placeholder="8位数字密钥，如 12345678" style="flex: 1" />
              <el-button type="success" @click="showCrackDialog" :disabled="cracking">一键破解</el-button>
            </div>
          </el-form-item>

          <el-form-item label="客户端IP" prop="iptv.ip">
            <el-input v-model="form.iptv.ip" placeholder="手动填写IP 或 填写网络接口名称(如 eth0, eth1.50)" />
            <div style="color: #909399; font-size: 12px; line-height: 1.4; margin-top: 4px">
              可填写具体IP地址，也可填写系统网络接口名称（程序自动获取该接口的IP）
            </div>
          </el-form-item>

          <el-divider content-position="left">自定义HTTP请求头</el-divider>
          <div style="margin-bottom: 16px">
            <div v-for="(header, idx) in form.iptv.headersList" :key="idx" style="display: flex; gap: 8px; margin-bottom: 8px">
              <el-input v-model="header.name" placeholder="Header名称" style="width: 200px" />
              <el-input v-model="header.value" placeholder="Header值" style="flex: 1" />
              <el-button :icon="Delete" circle size="small" @click="form.iptv.headersList.splice(idx, 1)" />
            </div>
            <el-button size="small" @click="form.iptv.headersList.push({ name: '', value: '' })">
              <el-icon><Plus /></el-icon> 添加请求头
            </el-button>
          </div>

          <el-divider content-position="left">认证接口参数</el-divider>
          <el-form-item label="认证参数" prop="iptv.authParamsStr">
            <el-input v-model="form.iptv.authParamsStr" type="textarea" :rows="10"
              placeholder="JSON格式，用于 ValidAuthenticationHW 接口的参数" />
            <div style="color: #909399; font-size: 12px; line-height: 1.6; margin-top: 4px">
              填写 ValidAuthenticationHW{{ form.iptv.providerSuffix }}.jsp 接口所需的参数（JSON格式）。
              <el-link type="primary" :underline="false" @click="fillAuthExample" style="font-size: 12px">填入示例配置</el-link>
            </div>
          </el-form-item>
        </template>

        <!-- Cron -->
        <el-form-item label="定时刷新" v-if="form.type !== 'network_manual'">
          <el-select v-model="form.cron_time" clearable placeholder="不定时刷新" style="width: 100%">
            <el-option v-for="opt in cronOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>

        <!-- EPG sync -->
        <template v-if="form.type === 'iptv' && !isEdit">
          <el-form-item label="同步EPG">
            <el-switch v-model="form.epg_enabled" />
            <span style="margin-left: 8px; color: #909399; font-size: 12px">自动创建关联的EPG源</span>
          </el-form-item>
          <el-form-item label="EPG策略" v-if="form.epg_enabled">
            <el-select v-model="form.iptv.channelProgramAPI" style="width: 100%">
              <el-option v-for="opt in epgStrategies" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <div style="color: #909399; font-size: 12px; margin-top: 4px">
              选择"自动检测"将依次尝试所有策略，成功后自动记录
            </div>
          </el-form-item>
        </template>

        <!-- Status (edit only) -->
        <el-form-item label="状态" v-if="isEdit">
          <el-switch v-model="form.status" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- Crack Key Dialog -->
    <el-dialog v-model="crackDialogVisible" title="一键破解密钥" width="500px" destroy-on-close :close-on-click-modal="false">
      <el-form label-width="100px">
        <el-form-item label="Authenticator">
          <el-input v-model="crackAuthenticator" type="textarea" :rows="3"
            placeholder="请粘贴抓包获取的 Authenticator 值（16进制字符串）" />
        </el-form-item>
        <el-alert v-if="crackResult" :title="'破解成功，密钥为: ' + crackResult" type="success" show-icon :closable="false" style="margin-bottom: 12px" />
        <el-alert v-if="crackError" :title="crackError" type="error" show-icon :closable="false" style="margin-bottom: 12px" />
      </el-form>
      <template #footer>
        <el-button @click="crackDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="doCrack" :loading="cracking">
          {{ cracking ? '破解中...' : '开始破解' }}
        </el-button>
        <el-button v-if="crackResult" type="success" @click="applyCrackResult">应用密钥</el-button>
      </template>
    </el-dialog>

    <!-- Channels Dialog -->
    <el-dialog v-model="channelsVisible" title="已解析频道列表" width="800px" destroy-on-close :close-on-click-modal="false">
      <p style="margin: 0 0 12px; color: #909399">共 {{ channels.length }} 个频道</p>
      <el-table :data="channels" max-height="400" border stripe size="small">
        <el-table-column prop="tvg_id" label="频道ID" width="120" show-overflow-tooltip />
        <el-table-column prop="name" label="频道名" width="130" />
        <el-table-column prop="group" label="分组" width="100" />
        <el-table-column label="地址" min-width="250">
          <template #default="{ row }">
            <div v-if="row.url && row.url.includes('|')">
              <div v-for="(u, i) in row.url.split('|')" :key="i" style="margin-bottom: 2px">
                <el-tag size="small" :type="u.startsWith('igmp://') || u.startsWith('rtp://') ? 'warning' : 'success'" style="margin-right: 4px">
                  {{ u.startsWith('igmp://') || u.startsWith('rtp://') ? '组播' : '单播' }}
                </el-tag>
                <span style="font-size: 12px; word-break: break-all">{{ u }}</span>
              </div>
            </div>
            <span v-else style="font-size: 12px; word-break: break-all">{{ row.url }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { List, Refresh, Edit, Delete, Plus, SuccessFilled, CircleCloseFilled } from '@element-plus/icons-vue'
import api from '../api'

const sources = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const channelsVisible = ref(false)
const channels = ref([])
const isEdit = ref(false)
const editId = ref(null)
const submitting = ref(false)
const formRef = ref()
const cronOptions = ref([])
const epgStrategies = ref([])

// Crack dialog state
const crackDialogVisible = ref(false)
const crackAuthenticator = ref('')
const crackResult = ref('')
const crackError = ref('')
const cracking = ref(false)

const typeNameMap = { iptv: 'IPTV', network_url: '网络URL', network_manual: '手动输入' }
const typeTagMap = { iptv: 'danger', network_url: '', network_manual: 'warning' }

const defaultHeaders = [
  { name: 'Accept', value: 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8' },
  { name: 'User-Agent', value: 'Mozilla/5.0 (X11; Linux x86_64; Fhbw2.0) AppleWebKit' },
  { name: 'Accept-Language', value: 'zh-CN,en-US;q=0.8' },
  { name: 'X-Requested-With', value: 'com.fiberhome.iptv' },
]

// Example auth params JSON: keys must match ValidAuthenticationHW API precisely
const authParamsExample = JSON.stringify({
  "UserID": "",
  "Lang": "",
  "SupportHD": "1",
  "NetUserID": "",
  "STBType": "",
  "STBVersion": "",
  "conntype": "",
  "STBID": "",
  "templateName": "",
  "areaId": "",
  "userGroupId": "",
  "productPackageId": "",
  "mac": "",
  "UserField": "",
  "SoftwareVersion": "",
  "IsSmartStb": "",
  "VIP": "",
  "desktopId": "",
  "stbmaker": "",
  "XMPPCapability": "",
  "ChipID": ""
}, null, 2)

const defaultForm = () => ({
  name: '', description: '', type: 'network_url', url: '', content: '', cron_time: '',
  epg_enabled: false, status: true,
  iptv: {
    platform: 'huawei',
    serverHost: '',
    providerSuffix: 'CTC',
    key: '',
    ip: '',
    channelProgramAPI: 'auto',
    headersList: defaultHeaders.map(h => ({ ...h })),
    authParamsStr: '',
  },
})
const form = reactive(defaultForm())

const formRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  url: [{ required: true, message: '请输入订阅URL', trigger: 'blur' }],
  content: [{ required: true, message: '请输入内容', trigger: 'blur' }],
  'iptv.platform': [{ required: true, message: '请选择IPTV平台', trigger: 'change' }],
  'iptv.serverHost': [{ required: true, message: '请输入IPTV服务器地址', trigger: 'blur' }],
  'iptv.providerSuffix': [{ required: true, message: '请选择运营商', trigger: 'change' }],
  'iptv.key': [{ required: true, message: '请输入加密密钥', trigger: 'blur' }],
  'iptv.ip': [{ required: true, message: '请输入客户端IP或接口名称', trigger: 'blur' }],
  'iptv.authParamsStr': [{ required: true, message: '请输入认证参数', trigger: 'blur' }],
}

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
    const { data } = await api.get('/live-sources')
    sources.value = data
  } finally { loading.value = false }
}

function onTypeChange() {
  // Reset IPTV fields when switching type
}

// Strip http:// or https:// prefix from serverHost, keep only host:port
function normalizeServerHost(input) {
  let host = input.trim()
  host = host.replace(/^https?:\/\//i, '')
  host = host.replace(/\/+$/, '') // Remove trailing slashes
  return host
}

function buildIptvConfig() {
  const iptv = form.iptv
  // Parse auth params JSON
  let authParams = {}
  if (iptv.authParamsStr.trim()) {
    try {
      authParams = JSON.parse(iptv.authParamsStr)
    } catch {
      throw new Error('认证参数JSON格式不正确')
    }
  }

  // Build headers map from list
  const headers = {}
  for (const h of iptv.headersList) {
    if (h.name.trim()) {
      headers[h.name.trim()] = h.value
    }
  }

  // Determine IP vs interfaceName
  const ipStr = iptv.ip.trim()
  let ip = ''
  let interfaceName = ''
  if (/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(ipStr)) {
    ip = ipStr
  } else if (ipStr) {
    interfaceName = ipStr
  }

  // Convert null values from authParams to empty strings for Go
  const str = (v) => (v === null || v === undefined) ? '' : String(v)

  return {
    platform: iptv.platform,
    serverHost: normalizeServerHost(iptv.serverHost),
    providerSuffix: iptv.providerSuffix,
    key: iptv.key,
    ip: ip,
    interfaceName: interfaceName,
    channelProgramAPI: iptv.channelProgramAPI || 'auto',
    headers: headers,
    // Pass the entire AuthParams JSON directly to the backend
    authParams: authParams,
  }
}

function parseIptvConfig(configStr) {
  let cfg = {}
  try {
    cfg = JSON.parse(configStr)
  } catch {}

  const headersList = []
  if (cfg.headers) {
    for (const [k, v] of Object.entries(cfg.headers)) {
      headersList.push({ name: k, value: v })
    }
  }

  // Restore the dynamic authParams back to the string block
  const authParamsStr = cfg.authParams ? JSON.stringify(cfg.authParams, null, 2) : authParamsExample

  let ipDisplay = cfg.ip || ''
  if (!ipDisplay && cfg.interfaceName) {
    ipDisplay = cfg.interfaceName
  }

  return {
    platform: cfg.platform || 'huawei',
    serverHost: cfg.serverHost || '',
    providerSuffix: cfg.providerSuffix || 'CTC',
    key: cfg.key || '',
    ip: ipDisplay,
    channelProgramAPI: cfg.channelProgramAPI || 'auto',
    headersList,
    authParamsStr,
  }
}

function showCreate() {
  isEdit.value = false
  editId.value = null
  Object.assign(form, defaultForm())
  dialogVisible.value = true
}

function showEdit(row) {
  isEdit.value = true
  editId.value = row.id
  const iptvParsed = row.type === 'iptv' ? parseIptvConfig(row.iptv_config) : defaultForm().iptv
  Object.assign(form, {
    name: row.name, description: row.description || '', type: row.type, url: row.url, content: row.content,
    cron_time: row.cron_time, status: row.status,
    iptv: iptvParsed,
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value.validate()
  submitting.value = true
  try {
    if (isEdit.value) {
      const body = { name: form.name, description: form.description, url: form.url, content: form.content, cron_time: form.cron_time, status: form.status }
      if (form.type === 'iptv') {
        body.iptv_config = buildIptvConfig()
      }
      await api.put(`/live-sources/${editId.value}`, body)
      ElMessage.success('更新成功')
    } else {
      const body = { name: form.name, description: form.description, type: form.type, url: form.url, content: form.content, cron_time: form.cron_time, epg_enabled: form.epg_enabled }
      if (form.type === 'iptv') {
        body.iptv_config = buildIptvConfig()
      }
      await api.post('/live-sources', body)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await loadSources()
  } catch (e) {
    if (e.message) ElMessage.error(e.message)
  }
  finally { submitting.value = false }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(
    `确定删除直播源 "${row.name}"？关联的EPG源和频道数据也将被删除。`,
    '确认删除',
    { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' }
  )
  await api.delete(`/live-sources/${row.id}`)
  ElMessage.success('删除成功')
  await loadSources()
}

async function triggerFetch(row) {
  await api.post(`/live-sources/${row.id}/trigger`)
  ElMessage.success('已触发刷新')
}

async function showChannels(row) {
  try {
    const { data } = await api.get(`/live-sources/${row.id}/channels`)
    channels.value = data.channels || []
    channelsVisible.value = true
  } catch {}
}

function fillAuthExample() {
  form.iptv.authParamsStr = authParamsExample
}

// Crack key
function showCrackDialog() {
  crackAuthenticator.value = ''
  crackResult.value = ''
  crackError.value = ''
  crackDialogVisible.value = true
}

async function doCrack() {
  if (!crackAuthenticator.value.trim()) {
    ElMessage.warning('请输入 Authenticator 值')
    return
  }
  cracking.value = true
  crackResult.value = ''
  crackError.value = ''
  try {
    const { data } = await api.post('/crack-key', { authenticator: crackAuthenticator.value.trim() }, { timeout: 360000 })
    crackResult.value = data.key
  } catch (e) {
    crackError.value = e.response?.data?.error || '破解失败'
  } finally {
    cracking.value = false
  }
}

function applyCrackResult() {
  if (crackResult.value) {
    form.iptv.key = crackResult.value
    crackDialogVisible.value = false
    ElMessage.success('密钥已应用')
  }
}
</script>
