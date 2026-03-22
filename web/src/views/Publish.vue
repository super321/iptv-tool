<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <h3 style="margin: 0">{{ $t('publish.title') }}</h3>
      <el-button type="primary" @click="showCreate">{{ $t('publish.add') }}</el-button>
    </div>
    <el-table :data="interfaces" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" :label="$t('common.name')" min-width="140" show-overflow-tooltip />
      <el-table-column prop="description" :label="$t('common.description')" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>
      <el-table-column prop="path" :label="$t('publish.col_path')" min-width="150">
        <template #default="{ row }">
          <el-link type="primary" :href="`/sub/${row.type}/${row.path}`" target="_blank">/sub/{{ row.type }}/{{ row.path }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="type" :label="$t('common.type')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.type === 'live' ? '' : 'success'" size="small">{{ row.type === 'live' ? $t('publish.type_live') : $t('publish.type_epg') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="format" :label="$t('publish.col_format')" width="100">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ row.format }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" :label="$t('common.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'" size="small">{{ row.status ? $t('common.enabled') : $t('common.disabled') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('publish.col_rule_count')" width="120" align="center">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ parseIds(row.rule_ids).length }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.operations')" width="120" fixed="right" align="center">
        <template #default="{ row }">
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
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('publish.edit_title') : $t('publish.add_title')" width="640px" destroy-on-close :close-on-click-modal="false">
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="auto">
        <el-tabs v-model="activeTab">
          <!-- Tab 1: Basic Info -->
          <el-tab-pane :label="$t('publish.tab_basic')" name="basic">
            <el-form-item :label="$t('common.name')" prop="name">
              <el-input v-model.trim="form.name" />
            </el-form-item>
            <el-form-item :label="$t('common.description')">
              <el-input v-model.trim="form.description" :placeholder="$t('publish.desc_placeholder')" />
            </el-form-item>
            <el-form-item :label="$t('publish.col_path')" prop="path">
              <el-input v-model.trim="form.path" placeholder="my_list">
                <template #prepend>/sub/{{ form.type }}/</template>
              </el-input>
            </el-form-item>
            <el-form-item :label="$t('common.type')" prop="type" v-if="!isEdit">
              <el-radio-group v-model="form.type" @change="onTypeChange">
                <el-radio value="live">{{ $t('publish.type_live') }}</el-radio>
                <el-radio value="epg">{{ $t('publish.type_epg') }}</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item :label="$t('publish.col_format')" prop="format">
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
          </el-tab-pane>

          <!-- Tab 2: Data Sources & Rules -->
          <el-tab-pane :label="$t('publish.tab_data')" name="data">
            <el-form-item :label="$t('publish.data_source')" prop="source_ids_arr">
              <el-select v-model="form.source_ids_arr" multiple :placeholder="$t('publish.select_source_placeholder')" style="width: 100%" @change="onSourceChange">
                <el-option v-for="src in availableSources" :key="src.id" :label="src.name" :value="src.id">
                  <span style="float: left">{{ src.name }}</span>
                  <span style="float: right; color: #8492a6; font-size: 13px" v-if="src.description">{{ src.description }}</span>
                </el-option>
              </el-select>
            </el-form-item>

            <!-- 已选直播数据源的过滤无效数据开关 -->
            <el-form-item v-if="form.type === 'live' && form.source_ids_arr.length > 0" :label="$t('publish.filter_invalid')">
              <div style="width: 100%">
                <div style="color: #909399; font-size: 12px; margin-bottom: 8px; line-height: 1.4">
                  {{ $t('publish.filter_invalid_help') }}
                </div>
                <div v-for="srcId in form.source_ids_arr" :key="srcId"
                  style="display: flex; justify-content: space-between; align-items: center; padding: 6px 12px; margin-bottom: 4px; background: var(--el-fill-color-light); border-radius: 4px;">
                  <span style="font-size: 13px;">{{ getSourceName(srcId) }}</span>
                  <el-switch
                    :model-value="form.filter_invalid_source_ids_arr.includes(srcId)"
                    @change="(val) => toggleFilterInvalid(srcId, val)"
                    size="small"
                  />
                </div>
              </div>
            </el-form-item>

            <el-form-item :label="$t('publish.agg_rules')" prop="rule_ids_arr">
              <el-select v-model="form.rule_ids_arr" multiple :placeholder="$t('publish.agg_rules_placeholder')" style="width: 100%">
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
            <el-form-item :label="$t('publish.tvg_id_mode')" v-if="(form.type === 'epg' && form.format === 'xmltv') || (form.type === 'live' && form.format === 'm3u')">
              <el-radio-group v-model="form.tvg_id_mode">
                <el-radio value="channel_id">{{ $t('publish.tvg_id_channel_id') }}</el-radio>
                <el-radio value="name">{{ $t('publish.tvg_id_name') }}</el-radio>
              </el-radio-group>
              <div style="color: #909399; font-size: 12px; line-height: 1.4; margin-top: 4px; width: 100%;">
                {{ $t('publish.tvg_id_help') }}
              </div>
            </el-form-item>
          </el-tab-pane>

          <!-- Tab 3: Output Settings -->
          <el-tab-pane :label="$t('publish.tab_output')" name="output">
            <template v-if="form.type === 'live'">
              <el-form-item :label="$t('publish.live_address')" prop="address_type">
                <el-select v-model="form.address_type" style="width: 100%">
                  <el-option :label="$t('publish.unicast_first')" value="unicast" />
                  <el-option :label="$t('publish.multicast_first')" value="multicast" />
                </el-select>
                <div style="color: #909399; font-size: 12px; margin-top: 4px; line-height: 1.4">
                  {{ $t('publish.address_type_help') }}
                </div>
              </el-form-item>

              <el-form-item :label="$t('publish.multicast_protocol')">
                <el-select v-model="form.multicast_type" style="width: 100%">
                  <el-option :label="$t('publish.udpxy_proxy')" value="udpxy" />
                  <el-option :label="$t('publish.igmp_direct')" value="igmp" />
                  <el-option :label="$t('publish.rtp_direct')" value="rtp" />
                </el-select>
              </el-form-item>

              <el-form-item :label="$t('publish.udpxy_address')" v-if="form.multicast_type === 'udpxy'" :rules="[{ required: true, message: $t('publish.udpxy_address_required'), trigger: 'blur' }]">
                <el-input v-model.trim="form.udpxy_url" :placeholder="$t('publish.udpxy_placeholder')" />
                <div style="color: #909399; font-size: 12px; line-height: 1.4; margin-top: 4px; width: 100%;">
                  {{ $t('publish.udpxy_help') }}
                </div>
              </el-form-item>

              <el-form-item :label="$t('publish.fcc_enable')" v-if="form.multicast_type === 'udpxy'">
                <el-switch v-model="form.fcc_enabled" />
                <div style="color: #909399; font-size: 12px; line-height: 1.4; margin-top: 4px; width: 100%;">
                  {{ $t('publish.fcc_enable_help') }}
                </div>
              </el-form-item>

              <el-form-item :label="$t('publish.fcc_type')" v-if="form.multicast_type === 'udpxy' && form.fcc_enabled">
                <el-select v-model="form.fcc_type" style="width: 100%">
                  <el-option :label="$t('publish.fcc_type_telecom')" value="telecom" />
                  <el-option :label="$t('publish.fcc_type_huawei')" value="huawei" />
                </el-select>
              </el-form-item>

              <el-form-item :label="$t('publish.catchup_template')" v-if="form.format === 'm3u'">
                <div style="width: 100%;">
                  <el-input v-model.trim="form.m3u_catchup_template" :placeholder="$t('publish.catchup_placeholder')" clearable>
                    <template #append>
                      <el-dropdown trigger="click" @command="(cmd) => form.m3u_catchup_template = cmd">
                        <span class="el-dropdown-link" style="cursor: pointer; display: flex; align-items: center; color: var(--el-color-primary)">
                          {{ $t('publish.select_template') }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                        </span>
                        <template #dropdown>
                          <el-dropdown-menu>
                            <el-dropdown-item command="playseek=${(b)yyyyMMddHHmmss}-${(e)yyyyMMddHHmmss}">{{ $t('publish.template_iptv') }}</el-dropdown-item>
                            <el-dropdown-item command="playseek={utc:YmdHMS}-{utcend:YmdHMS}">{{ $t('publish.template_tivimate') }}</el-dropdown-item>
                          </el-dropdown-menu>
                        </template>
                      </el-dropdown>
                    </template>
                  </el-input>
                </div>
              </el-form-item>
            </template>
            <template v-if="form.type === 'epg'">
              <el-form-item :label="$t('publish.epg_days')">
                <el-input-number v-model="form.epg_days" :min="0" :max="7" placeholder="" controls-position="right" />
                <span style="margin-left: 10px; color: #909399; font-size: 12px;">{{ $t('publish.epg_days_help') }}</span>
              </el-form-item>
              <el-form-item :label="$t('publish.gzip')" v-if="form.format === 'xmltv'">
                <el-switch v-model="form.gzip_enabled" />
              </el-form-item>
            </template>
          </el-tab-pane>

          <!-- Tab 4: Access Control -->
          <el-tab-pane :label="$t('publish.tab_access')" name="access">
            <el-form-item :label="$t('publish.ua_check')">
              <div style="width: 100%">
                <el-switch v-model="form.ua_check_enabled" />
                <div style="color: #909399; font-size: 12px; line-height: 1.4; margin-top: 4px">
                  {{ $t('publish.ua_check_help') }}
                </div>
              </div>
            </el-form-item>
            <el-form-item :label="$t('publish.ua_allowed_values')" v-if="form.ua_check_enabled">
              <el-input
                v-model="form.ua_allowed_values_text"
                type="textarea"
                :rows="3"
                :placeholder="$t('publish.ua_allowed_placeholder')"
              />
            </el-form-item>

            <el-form-item :label="$t('common.status')" v-if="isEdit">
              <el-switch v-model="form.status" />
            </el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <div style="display: flex; justify-content: space-between; width: 100%">
          <!-- 新增的预览按钮 -->
          <el-button type="success" plain @click="handlePreview" :icon="View">{{ $t('publish.preview') }}</el-button>
          <div>
            <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('common.confirm') }}</el-button>
          </div>
        </div>
      </template>
    </el-dialog>
    <!-- Preview Dialog (预览弹窗) -->
    <el-dialog v-model="previewVisible" :title="$t('publish.preview_title')" width="900px" append-to-body>
      <el-table :data="previewData" v-loading="previewLoading" height="500px" border stripe size="small">
        <template v-if="form.type === 'live'">
          <el-table-column prop="TVGId" :label="$t('publish.col_channel_id')" width="140" show-overflow-tooltip />
          <el-table-column prop="Name" :label="$t('publish.col_original_name')" min-width="150" show-overflow-tooltip />
          <el-table-column prop="Alias" :label="$t('publish.col_alias')" min-width="140">
            <template #default="{ row }"><span style="color: var(--el-color-primary); font-weight: bold">{{ row.Alias || '-' }}</span></template>
          </el-table-column>
          <el-table-column prop="Group" :label="$t('publish.col_group')" min-width="120" />
          <el-table-column prop="Logo" :label="$t('publish.col_logo')" width="80" align="center">
            <template #default="{ row }">
              <el-image 
                v-if="row.Logo || row.SourceLogo" 
                :src="row.Logo || row.SourceLogo" 
                style="width: 24px; height: 24px; cursor: pointer" 
                fit="contain" 
                :preview-src-list="[row.Logo || row.SourceLogo]"
                :z-index="3000"
                preview-teleported
                hide-on-click-modal
              />
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="URL" :label="$t('publish.col_live_url')" min-width="180" show-overflow-tooltip />
        </template>

        <template v-else>
          <el-table-column prop="channel_id" :label="$t('publish.col_channel_id')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="original_name" :label="$t('publish.col_original_name')" min-width="150" show-overflow-tooltip />
          <el-table-column prop="alias" :label="$t('publish.col_alias')" min-width="150">
            <template #default="{ row }"><span style="color: var(--el-color-primary); font-weight: bold">{{ row.alias || '-' }}</span></template>
          </el-table-column>
          <el-table-column prop="program_count" :label="$t('publish.col_program_count')" width="130" align="center">
            <template #default="{ row }"><el-tag size="small">{{ row.program_count }}</el-tag></template>
          </el-table-column>
        </template>
      </el-table>
    </el-dialog>
  </div>
</template>
<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Delete, View, ArrowDown } from '@element-plus/icons-vue'
import api from '../api'

const { t } = useI18n()

const typeNameMap = computed(() => ({ alias: t('rules.type_alias'), filter: t('rules.type_filter'), group: t('rules.type_group') }))
const interfaces = ref([])
const availableSources = ref([])
const availableRules = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const activeTab = ref('basic')
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
  address_type: 'multicast', multicast_type: 'udpxy', udpxy_url: '', fcc_enabled: false, fcc_type: 'telecom',
  m3u_catchup_template: '',
  epg_days: 7, gzip_enabled: false, tvg_id_mode: 'channel_id', filter_invalid_source_ids_arr: [],
  ua_check_enabled: false, ua_allowed_values_text: ''
})
const form = reactive(defaultForm())
const formRules = computed(() => ({
  name: [{ required: true, message: t('publish.name_required'), trigger: 'blur' }],
  path: [{ required: true, message: t('publish.path_required'), trigger: 'blur' }],
  format: [{ required: true, message: t('publish.format_required'), trigger: 'change' }],
  source_ids_arr: [{ required: true, message: t('publish.source_required'), trigger: 'change', type: 'array', min: 1 }],
}))
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
  form.filter_invalid_source_ids_arr = []

  // 类型切换时，清洗掉已选择的不兼容规则 (比如原本选了分组，现在切到EPG，就把分组ID剔除掉)
  if (newType === 'epg') {
    const validRuleIds = filteredRules.value.map(r => r.id)
    form.rule_ids_arr = form.rule_ids_arr.filter(id => validRuleIds.includes(id))
  }

  fetchSources(newType)
}
function onSourceChange(newIds) {
  // 新选择的源默认加入过滤列表，已移除的源从过滤列表中清除
  const added = newIds.filter(id => !form.filter_invalid_source_ids_arr.includes(id))
  form.filter_invalid_source_ids_arr = form.filter_invalid_source_ids_arr.filter(id => newIds.includes(id))
  form.filter_invalid_source_ids_arr.push(...added)
}
function getSourceName(srcId) {
  const src = availableSources.value.find(s => s.id === srcId)
  return src ? src.name : t('publish.source_fallback', { id: srcId })
}
function toggleFilterInvalid(srcId, val) {
  if (val) {
    if (!form.filter_invalid_source_ids_arr.includes(srcId)) {
      form.filter_invalid_source_ids_arr.push(srcId)
    }
  } else {
    form.filter_invalid_source_ids_arr = form.filter_invalid_source_ids_arr.filter(id => id !== srcId)
  }
}
function showCreate() {
  isEdit.value = false; editId.value = null
  Object.assign(form, defaultForm())
  activeTab.value = 'basic'
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
    filter_invalid_source_ids_arr: parseIds(row.filter_invalid_source_ids),
    status: row.status,
    tvg_id_mode: row.tvg_id_mode || 'channel_id',
    address_type: row.address_type || 'multicast',
    multicast_type: row.multicast_type || '', udpxy_url: row.udpxy_url || '',
    fcc_enabled: row.fcc_enabled || false, fcc_type: row.fcc_type || 'telecom',
    m3u_catchup_template: row.m3u_catchup_template || '',
    epg_days: row.epg_days || null, gzip_enabled: row.gzip_enabled || false,
    ua_check_enabled: row.ua_check_enabled || false,
    ua_allowed_values_text: (row.ua_allowed_values || '').split(',').filter(v => v.trim()).join('\n'),
  })
  activeTab.value = 'basic'
  fetchSources(form.type)
  dialogVisible.value = true
}
async function handlePreview() {
  if (form.source_ids_arr.length === 0) {
    ElMessage.warning(t('publish.select_source_first'))
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
      udpxy_url: form.udpxy_url,
      fcc_enabled: form.fcc_enabled,
      fcc_type: form.fcc_type,
      filter_invalid_source_ids: form.filter_invalid_source_ids_arr.join(',')
    })
    previewData.value = data || []
  } catch (e) {
    ElMessage.error(t('publish.preview_failed'))
  } finally {
    previewLoading.value = false
  }
}
// Map form field props to the tab they belong to
const fieldTabMap = {
  name: 'basic', path: 'basic', type: 'basic', format: 'basic',
  source_ids_arr: 'data', rule_ids_arr: 'data',
  address_type: 'output',
}
async function handleSubmit() {
  try {
    await formRef.value.validate()
  } catch (errors) {
    // Auto-switch to the tab containing the first validation error
    if (errors && typeof errors === 'object') {
      const firstField = Object.keys(errors)[0]
      if (firstField && fieldTabMap[firstField]) {
        activeTab.value = fieldTabMap[firstField]
      }
    }
    return
  }
  submitting.value = true
  try {
    const body = {
      name: form.name, description: form.description, path: form.path, type: form.type, format: form.format,
      source_ids: form.source_ids_arr.join(','), 
      rule_ids: form.rule_ids_arr.join(','),
      filter_invalid_source_ids: form.filter_invalid_source_ids_arr.join(','),
      tvg_id_mode: form.tvg_id_mode,
      address_type: form.address_type,
      multicast_type: form.multicast_type,
      udpxy_url: form.udpxy_url, fcc_enabled: form.fcc_enabled, fcc_type: form.fcc_type,
      m3u_catchup_template: form.m3u_catchup_template,
      epg_days: form.epg_days || 0, gzip_enabled: form.gzip_enabled,
      ua_check_enabled: form.ua_check_enabled,
      ua_allowed_values: form.ua_allowed_values_text.split('\n').map(v => v.trim()).filter(v => v).join(','),
    }
    if (isEdit.value) {
      body.status = form.status
      await api.put(`/publish/${editId.value}`, body)
      ElMessage.success(t('common.update_success'))
    } else {
      await api.post('/publish', body)
      ElMessage.success(t('common.create_success'))
    }
    dialogVisible.value = false
    await loadInterfaces()
  } catch {}
  finally { submitting.value = false }
}
async function handleDelete(row) {
  await ElMessageBox.confirm(t('publish.delete_confirm', { name: row.name }), t('common.confirm_delete'), { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') })
  await api.delete(`/publish/${row.id}`)
  ElMessage.success(t('common.delete_success'))
  await loadInterfaces()
}
</script>
