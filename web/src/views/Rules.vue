<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <h3 style="margin: 0">{{ $t('rules.title') }}</h3>
      <el-button type="primary" @click="showCreate">{{ $t('rules.add') }}</el-button>
    </div>

    <el-table :data="rules" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" :label="$t('rules.col_rule_name')" min-width="150" show-overflow-tooltip />
      <el-table-column prop="description" :label="$t('common.description')" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>
      <el-table-column prop="type" :label="$t('common.type')" width="150">
        <template #default="{ row }">
          <el-tag :type="typeTagMap[row.type]" size="small">{{ typeNameMap[row.type] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('rules.col_rule_count')" width="120" align="center">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ getRuleCount(row.config, row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" :label="$t('common.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'" size="small">{{ row.status ? $t('common.enabled') : $t('common.disabled') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.operations')" width="140" fixed="right" align="center">
        <template #default="{ row }">
          <el-tooltip :content="$t('common.edit')" placement="top" :show-after="500">
            <el-button icon="Edit" size="small" circle type="primary" @click="showEdit(row)" />
          </el-tooltip>
          <el-tooltip :content="$t('common.delete')" placement="top" :show-after="500">
            <el-button icon="Delete" size="small" circle type="danger" @click="handleDelete(row)" />
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create/Edit Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('rules.edit_title') : $t('rules.add_title')" width="700px" destroy-on-close :close-on-click-modal="false">
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item :label="$t('rules.col_rule_name')" prop="name">
          <el-input v-model.trim="form.name" :placeholder="$t('rules.rule_name_placeholder')" />
        </el-form-item>
        <el-form-item :label="$t('common.description')" prop="description">
          <el-input v-model.trim="form.description" :placeholder="$t('rules.rule_desc_placeholder')" />
        </el-form-item>
        <el-form-item :label="$t('common.type')" prop="type" v-if="!isEdit">
          <el-radio-group v-model="form.type" @change="onTypeChange">
            <el-radio value="alias">{{ $t('rules.type_alias') }}</el-radio>
            <el-radio value="filter">{{ $t('rules.type_filter') }}</el-radio>
            <el-radio value="group">{{ $t('rules.type_group') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item :label="$t('common.status')" v-if="isEdit">
          <el-switch v-model="form.status" />
        </el-form-item>

        <!-- Dynamic Rule Configs based on Type -->
        <el-divider>{{ $t('rules.config_detail') }}</el-divider>
        
        <!-- Type: Alias -->
        <template v-if="form.type === 'alias'">
          <div style="margin-bottom: 8px; color: #909399; font-size: 13px;">
            {{ $t('rules.alias_help') }}
          </div>
          <div v-for="(rule, idx) in aliasConfig" :key="idx" class="rule-box">
            <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px;">
              <span style="font-weight: bold; color: var(--el-text-color-regular); font-size: 14px;">{{ $t('rules.alias_rule_n', { n: idx + 1 }) }}</span>
              <el-button size="small" type="danger" link @click="aliasConfig.splice(idx, 1)">{{ $t('common.delete') }}</el-button>
            </div>
            <el-row :gutter="10">
              <el-col :span="6">
                <el-select v-model="rule.match_mode" size="small" :placeholder="$t('rules.match_mode_placeholder')">
                  <el-option :label="$t('rules.regex')" value="regex" />
                  <el-option :label="$t('rules.string')" value="string" />
                </el-select>
              </el-col>
              <el-col :span="9">
                <el-input v-model="rule.pattern" size="small" :placeholder="$t('rules.match_content_placeholder')" />
              </el-col>
              <el-col :span="9">
                <el-input v-model="rule.replacement" size="small" :placeholder="$t('rules.replacement_placeholder')" />
              </el-col>
            </el-row>
          </div>
          <el-button size="small" style="width: 100%" @click="aliasConfig.push({ match_mode: 'regex', pattern: '', replacement: '' })">
            {{ $t('rules.add_alias_rule') }}
          </el-button>
        </template>

        <!-- Type: Filter -->
        <template v-if="form.type === 'filter'">
          <div style="margin-bottom: 8px; color: #909399; font-size: 13px;">
            {{ $t('rules.filter_help') }}
          </div>
          <div v-for="(rule, idx) in filterConfig" :key="idx" class="rule-box">
            <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px;">
              <span style="font-weight: bold; color: var(--el-text-color-regular); font-size: 14px;">{{ $t('rules.filter_rule_n', { n: idx + 1 }) }}</span>
              <el-button size="small" type="danger" link @click="filterConfig.splice(idx, 1)">{{ $t('common.delete') }}</el-button>
            </div>
            <el-row :gutter="10">
              <el-col :span="6">
                <el-select v-model="rule.target" size="small" :placeholder="$t('rules.match_target_placeholder')">
                  <el-option :label="$t('rules.target_name')" value="name" />
                  <el-option :label="$t('rules.target_alias')" value="alias" />
                </el-select>
              </el-col>
              <el-col :span="6">
                <el-select v-model="rule.match_mode" size="small" :placeholder="$t('rules.match_mode_placeholder')">
                  <el-option :label="$t('rules.regex')" value="regex" />
                  <el-option :label="$t('rules.string')" value="string" />
                </el-select>
              </el-col>
              <el-col :span="12">
                <el-input v-model="rule.pattern" size="small" :placeholder="$t('rules.filter_content_placeholder')" />
              </el-col>
            </el-row>
          </div>
          <el-button size="small" style="width: 100%" @click="filterConfig.push({ target: 'name', match_mode: 'regex', pattern: '' })">
            {{ $t('rules.add_filter_rule') }}
          </el-button>
        </template>

        <!-- Type: Group -->
        <template v-if="form.type === 'group'">
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
            <span style="color: #909399; font-size: 13px;">{{ $t('rules.group_help') }}</span>
            <el-button size="small" type="success" @click="startAIGenerate" icon="MagicStick">
              {{ $t('rules.ai_generate') }}
            </el-button>
          </div>
          <div v-for="(group, gIdx) in groupConfig" :key="gIdx" class="rule-box" style="background: var(--rule-box-group-bg)">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px">
              <el-input v-model="group.group_name" size="small" :placeholder="$t('rules.group_name_placeholder')" style="width: 200px" />
              <el-button size="small" type="danger" link @click="groupConfig.splice(gIdx, 1)">{{ $t('rules.delete_group') }}</el-button>
            </div>
            
            <div v-for="(rule, rIdx) in group.rules" :key="rIdx" style="margin-bottom: 6px; display: flex; gap: 8px">
              <el-select v-model="rule.target" size="small" style="width: 100px" :placeholder="$t('rules.match_target_placeholder')">
                <el-option :label="$t('rules.target_name')" value="name" />
                <el-option :label="$t('rules.target_alias')" value="alias" />
              </el-select>
              <el-select v-model="rule.match_mode" size="small" style="width: 110px">
                <el-option :label="$t('rules.regex')" value="regex" />
                <el-option :label="$t('rules.string')" value="string" />
              </el-select>
              <el-input v-model="rule.pattern" size="small" :placeholder="$t('rules.match_content_short')" style="flex: 1" />
              <el-button size="small" icon="Delete" circle @click="group.rules.splice(rIdx, 1)" />
            </div>
            <el-button size="small" type="primary" plain @click="group.rules.push({ target: 'name', match_mode: 'regex', pattern: '' })">
              {{ $t('rules.add_match_condition') }}
            </el-button>
          </div>
          <el-button size="small" style="width: 100%" @click="groupConfig.push({ group_name: '', rules: [{ target: 'name', match_mode: 'regex', pattern: '' }] })">
            {{ $t('rules.add_group') }}
          </el-button>
        </template>

      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- AI Step 1: Source Selection Dialog -->
    <el-dialog v-model="aiSourceDialogVisible" :title="$t('rules.ai_select_sources')" width="550px" destroy-on-close :close-on-click-modal="false">
      <p style="color: #909399; font-size: 13px; margin-top: 0;">{{ $t('rules.ai_select_sources_desc') }}</p>
      <div v-if="aiSourcesLoading" v-loading="true" style="height: 120px"></div>
      <template v-else>
        <el-empty v-if="aiSources.length === 0" :description="$t('rules.ai_no_sources')" />
        <el-checkbox-group v-model="aiSelectedSourceIds" v-else>
          <div v-for="src in aiSources" :key="src.id" style="margin-bottom: 8px;">
            <el-checkbox :value="src.id">
              {{ src.name }}
              <el-tag size="small" type="info" style="margin-left: 6px;">{{ src.channel_count }} {{ $t('live_sources.channel_count') }}</el-tag>
            </el-checkbox>
          </div>
        </el-checkbox-group>
      </template>
      <template #footer>
        <el-button @click="aiSourceDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="aiLoadChannelsAndBuildPrompt" :loading="aiChannelsLoading" :disabled="aiSelectedSourceIds.length === 0">
          {{ $t('rules.ai_next') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- AI Step 2: Prompt Display Dialog -->
    <el-dialog v-model="aiPromptDialogVisible" :title="$t('rules.ai_prompt_ready')" width="700px" destroy-on-close :close-on-click-modal="false">
      <el-alert :title="$t('rules.ai_prompt_instruction')" type="info" :closable="false" show-icon style="margin-bottom: 12px;" />
      <el-tag type="success" size="small" style="margin-bottom: 8px;">{{ $t('rules.ai_channel_count', { count: aiChannelNames.length }) }}</el-tag>
      <el-input v-model="aiPromptText" type="textarea" :rows="14" readonly style="font-family: monospace; font-size: 12px;" />
      <template #footer>
        <el-button @click="aiPromptDialogVisible = false; aiSourceDialogVisible = true">{{ $t('rules.ai_prev') }}</el-button>
        <el-button type="primary" @click="aiCopyAndNext" icon="DocumentCopy">{{ $t('rules.ai_copy_prompt') }}</el-button>
      </template>
    </el-dialog>

    <!-- AI Step 3: Response Input Dialog -->
    <el-dialog v-model="aiResponseDialogVisible" :title="$t('rules.ai_paste_response')" width="700px" destroy-on-close :close-on-click-modal="false">
      <el-input v-model="aiResponseText" type="textarea" :rows="14" :placeholder="$t('rules.ai_paste_placeholder')" style="font-family: monospace; font-size: 12px;" />
      <template #footer>
        <el-button @click="aiResponseDialogVisible = false; aiPromptDialogVisible = true">{{ $t('rules.ai_prev') }}</el-button>
        <el-button type="primary" @click="aiParseResponse">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, MagicStick, DocumentCopy } from '@element-plus/icons-vue'
import api from '../api'
import { getPromptTemplate, validateGroupRulesJSON } from '../utils/promptTemplates'

const { t } = useI18n()

const rules = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const submitting = ref(false)
const formRef = ref()

const typeNameMap = computed(() => ({ alias: t('rules.type_alias'), filter: t('rules.type_filter'), group: t('rules.type_group') }))
const typeTagMap = { alias: 'primary', filter: 'danger', group: 'warning' }

const form = reactive({ name: '', type: 'alias', status: true })
const formRules = computed(() => ({
  name: [{ required: true, message: t('rules.rule_name_placeholder'), trigger: 'blur' }],
}))

// Config states
const aliasConfig = ref([])
const filterConfig = ref([])
const groupConfig = ref([])

// --- AI Generate states ---
const aiSourceDialogVisible = ref(false)
const aiPromptDialogVisible = ref(false)
const aiResponseDialogVisible = ref(false)
const aiSources = ref([])
const aiSourcesLoading = ref(false)
const aiSelectedSourceIds = ref([])
const aiChannelsLoading = ref(false)
const aiChannelNames = ref([])
const aiPromptText = ref('')
const aiResponseText = ref('')

onMounted(() => loadRules())

async function loadRules() {
  loading.value = true
  try {
    const { data } = await api.get('/rules')
    rules.value = data || []
  } finally { loading.value = false }
}

function getRuleCount(configStr, type) {
  try {
    const config = JSON.parse(configStr)
    if (!Array.isArray(config)) return 0
    if (type === 'group') {
      return config.reduce((acc, curr) => acc + (curr.rules ? curr.rules.length : 0), 0)
    }
    return config.length
  } catch (e) {
    return 0
  }
}

function onTypeChange() {
  aliasConfig.value = []
  filterConfig.value = []
  groupConfig.value = []
  if (form.type === 'alias') aliasConfig.value.push({ match_mode: 'regex', pattern: '', replacement: '' })
  if (form.type === 'filter') filterConfig.value.push({ target: 'name', match_mode: 'regex', pattern: '' })
  if (form.type === 'group') groupConfig.value.push({ group_name: '', rules: [{ target: 'name', match_mode: 'regex', pattern: '' }] })
}

function showCreate() {
  isEdit.value = false
  editId.value = null
  form.name = ''
  form.description = ''
  form.type = 'alias'
  form.status = true
  onTypeChange()
  dialogVisible.value = true
}

function showEdit(row) {
  isEdit.value = true
  editId.value = row.id
  form.name = row.name
  form.description = row.description || ''
  form.type = row.type
  form.status = row.status

  aliasConfig.value = []
  filterConfig.value = []
  groupConfig.value = []

  let parsed = []
  try { parsed = JSON.parse(row.config) } catch (e) { parsed = [] }

  if (row.type === 'alias') aliasConfig.value = parsed
  if (row.type === 'filter') filterConfig.value = parsed
  if (row.type === 'group') groupConfig.value = parsed

  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value.validate()
  
  let configData = []
  if (form.type === 'alias') {
    configData = aliasConfig.value.filter(r => r.pattern.trim())
  } else if (form.type === 'filter') {
    configData = filterConfig.value.filter(r => r.pattern.trim())
  } else if (form.type === 'group') {
    configData = groupConfig.value.filter(g => g.group_name.trim()).map(g => ({
      group_name: g.group_name,
      rules: g.rules.filter(r => r.pattern.trim())
    })).filter(g => g.rules.length > 0)
  }

  if (configData.length === 0) {
    ElMessage.warning(t('rules.config_empty'))
    return
  }

  submitting.value = true
  try {
    const body = {
      name: form.name,
      description: form.description,
      type: form.type,
      config: configData
    }
    
    if (isEdit.value) {
      body.status = form.status
      await api.put(`/rules/${editId.value}`, body)
      ElMessage.success(t('common.update_success'))
    } else {
      await api.post('/rules', body)
      ElMessage.success(t('common.create_success'))
    }
    dialogVisible.value = false
    await loadRules()
  } catch {}
  finally { submitting.value = false }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(t('rules.delete_confirm', { name: row.name }), t('common.confirm_delete'), { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') })
  await api.delete(`/rules/${row.id}`)
  ElMessage.success(t('common.delete_success'))
  await loadRules()
}

// ===========================
// AI Generate Workflow
// ===========================

async function startAIGenerate() {
  // Reset AI state
  aiSelectedSourceIds.value = []
  aiChannelNames.value = []
  aiPromptText.value = ''
  aiResponseText.value = ''
  aiSources.value = []

  // Show source selection dialog and load sources
  aiSourceDialogVisible.value = true
  aiSourcesLoading.value = true
  try {
    const { data } = await api.get('/live-sources')
    aiSources.value = (data || []).filter(s => s.channel_count > 0)
  } catch {
    aiSources.value = []
  } finally {
    aiSourcesLoading.value = false
  }
}

async function aiLoadChannelsAndBuildPrompt() {
  aiChannelsLoading.value = true
  try {
    // Fetch channels from each selected source in parallel
    const requests = aiSelectedSourceIds.value.map(id =>
      api.get(`/live-sources/${id}/channels`).then(res => res.data?.channels || [])
    )
    const results = await Promise.all(requests)

    // Collect unique channel names
    const nameSet = new Set()
    for (const channels of results) {
      for (const ch of channels) {
        if (ch.name) nameSet.add(ch.name)
      }
    }

    aiChannelNames.value = [...nameSet].sort()

    if (aiChannelNames.value.length === 0) {
      ElMessage.warning(t('rules.ai_no_channels'))
      return
    }

    // Build prompt using template
    const template = getPromptTemplate('group_rules')
    aiPromptText.value = template.build(aiChannelNames.value)

    // Move to prompt dialog
    aiSourceDialogVisible.value = false
    aiPromptDialogVisible.value = true

    // Try to auto-copy to clipboard
    try {
      await navigator.clipboard.writeText(aiPromptText.value)
      ElMessage.success(t('rules.ai_prompt_copied'))
    } catch {
      ElMessage.warning(t('rules.ai_prompt_copy_failed'))
    }
  } catch {
    ElMessage.error(t('rules.ai_no_channels'))
  } finally {
    aiChannelsLoading.value = false
  }
}

function aiCopyAndNext() {
  navigator.clipboard.writeText(aiPromptText.value).then(() => {
    ElMessage.success(t('rules.ai_prompt_copied'))
  }).catch(() => {
    ElMessage.warning(t('rules.ai_prompt_copy_failed'))
  })
  aiPromptDialogVisible.value = false
  aiResponseDialogVisible.value = true
}

async function aiParseResponse() {
  const text = aiResponseText.value.trim()
  if (!text) {
    ElMessage.warning(t('rules.ai_parse_empty'))
    return
  }

  const result = validateGroupRulesJSON(text)
  if (!result.valid) {
    ElMessage.error(t('rules.ai_parse_error'))
    return
  }

  // Directly overwrite existing group config with AI-generated rules
  groupConfig.value = result.data.map(g => ({
    group_name: g.group_name,
    rules: g.rules.map(r => ({
      target: r.target || 'name',
      match_mode: r.match_mode || 'regex',
      pattern: r.pattern
    }))
  }))

  aiResponseDialogVisible.value = false
  ElMessage.success(t('rules.ai_parse_success', { count: result.data.length }))
}
</script>

<style scoped>
.rule-box {
  border: 1px solid var(--rule-box-border);
  padding: 16px;
  border-radius: 4px;
  margin-bottom: 12px;
}
</style>
