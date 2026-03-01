<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <h3 style="margin: 0">聚合发布</h3>
      <el-button type="primary" @click="showCreate">新增发布接口</el-button>
    </div>
    <el-table :data="interfaces" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" width="130" show-overflow-tooltip />
      <el-table-column prop="description" label="描述" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>
      <el-table-column prop="path" label="路径" min-width="120">
        <template #default="{ row }">
          <el-link type="primary" :href="`/sub/${row.type}/${row.path}`" target="_blank">/sub/{{ row.type }}/{{ row.path }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="80">
        <template #default="{ row }">
          <el-tag :type="row.type === 'live' ? '' : 'success'" size="small">{{ row.type === 'live' ? '直播' : 'EPG' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="format" label="格式" width="80">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ row.format }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'" size="small">{{ row.status ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="应用规则数" width="100" align="center">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ parseIds(row.rule_ids).length }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right" align="center">
        <template #default="{ row }">
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
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑发布接口' : '新增发布接口'" width="600px" destroy-on-close :close-on-click-modal="false">
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" placeholder="可选的描述信息" />
        </el-form-item>
        <el-form-item label="路径" prop="path">
          <el-input v-model="form.path" placeholder="my_list">
            <template #prepend>/sub/{{ form.type }}/</template>
          </el-input>
        </el-form-item>
        <el-form-item label="类型" prop="type" v-if="!isEdit">
          <el-radio-group v-model="form.type" @change="onTypeChange">
            <el-radio value="live">直播</el-radio>
            <el-radio value="epg">EPG</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="格式" prop="format">
          <el-select v-model="form.format" style="width: 100%">
            <template v-if="form.type === 'live'">
              <el-option label="M3U" value="m3u" />
              <el-option label="TXT (DIYP)" value="txt" />
            </template>
            <template v-else>
              <el-option label="XMLTV" value="xmltv" />
              <el-option label="DIYP JSON" value="diyp" />
            </template>
          </el-select>
        </el-form-item>

        <el-form-item label="数据源" prop="source_ids_arr">
          <el-select v-model="form.source_ids_arr" multiple placeholder="请选择启用的数据源" style="width: 100%">
            <el-option v-for="src in availableSources" :key="src.id" :label="src.name" :value="src.id" />
          </el-select>
        </el-form-item>

        <!-- 新的聚合规则多选下拉框 -->
        <el-form-item label="聚合规则" prop="rule_ids_arr">
          <el-select v-model="form.rule_ids_arr" multiple placeholder="可选，按选择顺序执行匹配" style="width: 100%">
            <el-option 
              v-for="rule in filteredRules" 
              :key="rule.id" 
              :label="rule.name" 
              :value="rule.id"
            >
              <span style="float: left">{{ rule.name }}</span>
              <span style="float: right; color: #8492a6; font-size: 13px">{{ typeNameMap[rule.type] || rule.type }}</span>
            </el-option>
          </el-select>
        </el-form-item>

        <!-- 关联策略 (tvg-id) -->
        <el-form-item label="关联策略" v-if="(form.type === 'epg' && form.format === 'xmltv') || (form.type === 'live' && form.format === 'm3u')">
          <el-radio-group v-model="form.tvg_id_mode">
            <el-radio value="channel_id">使用原始频道ID (默认)</el-radio>
            <el-radio value="name">使用频道别名 / 名称</el-radio>
          </el-radio-group>
          <div style="color: #909399; font-size: 12px; line-height: 1.4; margin-top: 4px; width: 100%;">
            此项决定 M3U 的 "tvg-id" 或 XMLTV 的 "channel id" 取值。<br/>
            第三方播放器主要依赖此 ID 进行台标与节目的精准匹配。
          </div>
        </el-form-item>

        <template v-if="form.type === 'live'">
          <el-form-item label="直播地址" prop="address_type">
            <el-select v-model="form.address_type" style="width: 100%">
              <el-option label="单播优先" value="unicast" />
              <el-option label="组播优先 (默认)" value="multicast" />
            </el-select>
            <div style="color: #909399; font-size: 12px; margin-top: 4px; line-height: 1.4">
              单播优先：提取单播地址(HTTP/RTSP等)，若无则尝试借用时移地址替补<br/>
              组播优先：优先提取组播地址(IGMP)，无组播时平滑降级使用单播
            </div>
          </el-form-item>

          <el-form-item label="组播协议" v-if="form.address_type === 'multicast'">
            <el-select v-model="form.multicast_type" style="width: 100%">
              <el-option label="UDPXY代理 (默认)" value="udpxy" />
              <el-option label="IGMP直连" value="igmp" />
              <el-option label="RTP直连" value="rtp" />
            </el-select>
          </el-form-item>

          <el-form-item label="UDPxy地址" v-if="form.address_type === 'multicast' && form.multicast_type === 'udpxy'" :rules="[{ required: true, message: '请填写UDPxy地址', trigger: 'blur' }]">
            <el-input v-model="form.udpxy_url" placeholder="例如: http://192.168.1.1:4022" />
          </el-form-item>

          <el-form-item label="回看模板" v-if="form.format === 'm3u'">
            <div style="width: 100%;">
              <el-input v-model="form.m3u_catchup_template" placeholder="选填，若需回看功能请填充相关模板参数" clearable>
                <template #append>
                  <el-dropdown trigger="click" @command="(cmd) => form.m3u_catchup_template = cmd">
                    <span class="el-dropdown-link" style="cursor: pointer; display: flex; align-items: center; color: var(--el-color-primary)">
                      选择模板<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                    </span>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="playseek=${(b)yyyyMMddHHmmss}-${(e)yyyyMMddHHmmss}">通用 IPTV (yyyyMMddHHmmss)</el-dropdown-item>
                        <el-dropdown-item command="playseek={utc:YmdHMS}-{utcend:YmdHMS}">通用 TiviMate (YmdHMS)</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </template>
              </el-input>
            </div>
          </el-form-item>
        </template>
        <template v-if="form.type === 'epg'">
          <el-form-item label="保留天数">
            <el-input-number v-model="form.epg_days" :min="0" :max="7" placeholder="空表示全部" controls-position="right" />
            <span style="margin-left: 10px; color: #909399; font-size: 12px;">(为0或空则不限制天数)</span>
          </el-form-item>
          <el-form-item label="Gzip压缩" v-if="form.format === 'xmltv'">
            <el-switch v-model="form.gzip_enabled" />
          </el-form-item>
        </template>
        <el-form-item label="状态" v-if="isEdit">
          <el-switch v-model="form.status" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; justify-content: space-between; width: 100%">
          <!-- 新增的预览按钮 -->
          <el-button type="success" plain @click="handlePreview" :icon="View">预览效果</el-button>
          <div>
            <el-button @click="dialogVisible = false">取消</el-button>
            <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
          </div>
        </div>
      </template>
    </el-dialog>
    <!-- Preview Dialog (预览弹窗) -->
    <el-dialog v-model="previewVisible" title="聚合预览" width="900px" append-to-body>
      <el-table :data="previewData" v-loading="previewLoading" height="500px" border stripe size="small">
        <template v-if="form.type === 'live'">
          <el-table-column prop="TVGId" label="频道ID" width="100" show-overflow-tooltip />
          <el-table-column prop="Name" label="原始名称" width="120" show-overflow-tooltip />
          <el-table-column prop="Alias" label="频道别名" width="120">
            <template #default="{ row }"><span style="color: #409eff; font-weight: bold">{{ row.Alias || '-' }}</span></template>
          </el-table-column>
          <el-table-column prop="Group" label="频道分组" width="100" />
          <el-table-column prop="Logo" label="台标" width="60" align="center">
            <template #default="{ row }">
              <el-image 
                v-if="row.Logo" 
                :src="row.Logo" 
                style="width: 24px; height: 24px; cursor: pointer" 
                fit="contain" 
                :preview-src-list="[row.Logo]"
                :z-index="3000"
                preview-teleported
                hide-on-click-modal
              />
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="URL" label="直播地址" min-width="180" show-overflow-tooltip />
        </template>

        <template v-else>
          <el-table-column prop="channel_id" label="频道ID" min-width="150" show-overflow-tooltip />
          <el-table-column prop="original_name" label="原始名称" width="150" show-overflow-tooltip />
          <el-table-column prop="alias" label="频道别名" width="150">
            <template #default="{ row }"><span style="color: #409eff; font-weight: bold">{{ row.alias || '-' }}</span></template>
          </el-table-column>
          <el-table-column prop="program_count" label="节目数" width="100" align="center">
            <template #default="{ row }"><el-tag size="small">{{ row.program_count }}</el-tag></template>
          </el-table-column>
        </template>
      </el-table>
    </el-dialog>
  </div>
</template>
<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Delete, View } from '@element-plus/icons-vue'
import api from '../api'
const typeNameMap = { alias: '频道别名', filter: '频道过滤', group: '频道分组' }
const interfaces = ref([])
const availableSources = ref([])
const availableRules = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const submitting = ref(false)
const formRef = ref()
// Preview state
const previewVisible = ref(false)
const previewLoading = ref(false)
const previewData = ref([])
const defaultForm = () => ({
  name: '', description: '', path: '', type: 'live', format: 'm3u', source_ids_arr: [], rule_ids_arr: [], status: true,
  address_type: 'multicast', multicast_type: 'udpxy', udpxy_url: '', m3u_catchup_template: '',
  epg_days: 7, gzip_enabled: false, tvg_id_mode: 'channel_id'
})
const form = reactive(defaultForm())
const formRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  path: [{ required: true, message: '请输入路径', trigger: 'blur' }],
  format: [{ required: true, message: '请选择格式', trigger: 'change' }],
  source_ids_arr: [{ required: true, message: '请至少选择一个数据源', trigger: 'change', type: 'array', min: 1 }],
}
// 动态计算下拉框内能选择的合法规则：EPG不支持选择"分组"
const filteredRules = computed(() => {
  if (form.type === 'epg') {
    return availableRules.value.filter(rule => rule.type !== 'group')
  }
  return availableRules.value
})
onMounted(() => {
  loadInterfaces()
  fetchRules()
})
async function loadInterfaces() {
  loading.value = true
  try {
    const { data } = await api.get('/publish')
    interfaces.value = data || []
  } finally { loading.value = false }
}
async function fetchSources(type) {
  try {
    const endpoint = type === 'live' ? '/live-sources' : '/epg-sources'
    const { data } = await api.get(endpoint)
    availableSources.value = (data || []).filter(s => s.status)
  } catch {}
}
async function fetchRules() {
  try {
    const { data } = await api.get('/rules')
    availableRules.value = (data || []).filter(r => r.status)
  } catch {}
}
function onTypeChange(newType) {
  form.format = newType === 'live' ? 'm3u' : 'xmltv'
  form.source_ids_arr = []

  // 类型切换时，清洗掉已选择的不兼容规则 (比如原本选了分组，现在切到EPG，就把分组ID剔除掉)
  if (newType === 'epg') {
    const validRuleIds = filteredRules.value.map(r => r.id)
    form.rule_ids_arr = form.rule_ids_arr.filter(id => validRuleIds.includes(id))
  }

  fetchSources(newType)
}
function showCreate() {
  isEdit.value = false; editId.value = null
  Object.assign(form, defaultForm())
  fetchSources(form.type)
  dialogVisible.value = true
}
function parseIds(str) {
  if (!str) return []
  return str.split(',').map(Number).filter(n => !isNaN(n))
}
function showEdit(row) {
  isEdit.value = true; editId.value = row.id
  Object.assign(form, {
    name: row.name, description: row.description || '', path: row.path, type: row.type, format: row.format,
    source_ids_arr: parseIds(row.source_ids),
    rule_ids_arr: parseIds(row.rule_ids),
    status: row.status,
    tvg_id_mode: row.tvg_id_mode || 'channel_id',
    address_type: row.address_type || 'multicast',
    multicast_type: row.multicast_type || '', udpxy_url: row.udpxy_url || '',
    m3u_catchup_template: row.m3u_catchup_template || '',
    epg_days: row.epg_days || null, gzip_enabled: row.gzip_enabled || false,
  })
  fetchSources(form.type)
  dialogVisible.value = true
}
async function handlePreview() {
  if (form.source_ids_arr.length === 0) {
    ElMessage.warning('请先选择至少一个数据源')
    return
  }

  previewVisible.value = true
  previewLoading.value = true
  previewData.value = []

  try {
    const { data } = await api.post('/publish/preview', {
      type: form.type,
      source_ids: form.source_ids_arr.join(','),
      rule_ids: form.rule_ids_arr.join(','),
      address_type: form.address_type,
      multicast_type: form.multicast_type,
      udpxy_url: form.udpxy_url
    })
    previewData.value = data || []
  } catch (e) {
    ElMessage.error('预览失败')
  } finally {
    previewLoading.value = false
  }
}
async function handleSubmit() {
  await formRef.value.validate()
  submitting.value = true
  try {
    const body = {
      name: form.name, description: form.description, path: form.path, type: form.type, format: form.format,
      source_ids: form.source_ids_arr.join(','), 
      rule_ids: form.rule_ids_arr.join(','),
      tvg_id_mode: form.tvg_id_mode,
      address_type: form.address_type,
      multicast_type: form.multicast_type,
      udpxy_url: form.udpxy_url, m3u_catchup_template: form.m3u_catchup_template,
      epg_days: form.epg_days || 0, gzip_enabled: form.gzip_enabled,
    }
    if (isEdit.value) {
      body.status = form.status
      await api.put(`/publish/${editId.value}`, body)
      ElMessage.success('更新成功')
    } else {
      await api.post('/publish', body)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await loadInterfaces()
  } catch {}
  finally { submitting.value = false }
}
async function handleDelete(row) {
  await ElMessageBox.confirm(`确定删除发布接口 "${row.name}"？`, '确认删除', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' })
  await api.delete(`/publish/${row.id}`)
  ElMessage.success('删除成功')
  await loadInterfaces()
}
</script>
