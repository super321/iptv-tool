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
      <el-table-column prop="type" :label="$t('common.type')" width="120">
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
          <el-input v-model="form.name" :placeholder="$t('rules.rule_name_placeholder')" />
        </el-form-item>
        <el-form-item :label="$t('common.description')" prop="description">
          <el-input v-model="form.description" :placeholder="$t('rules.rule_desc_placeholder')" />
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
              <span style="font-weight: bold; color: #606266; font-size: 14px;">{{ $t('rules.alias_rule_n', { n: idx + 1 }) }}</span>
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
              <span style="font-weight: bold; color: #606266; font-size: 14px;">{{ $t('rules.filter_rule_n', { n: idx + 1 }) }}</span>
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
          <div style="margin-bottom: 8px; color: #909399; font-size: 13px;">
            {{ $t('rules.group_help') }}
          </div>
          <div v-for="(group, gIdx) in groupConfig" :key="gIdx" class="rule-box" style="background: #fafafa">
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
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit } from '@element-plus/icons-vue'
import api from '../api'

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
</script>

<style scoped>
.rule-box {
  border: 1px solid #ebeef5;
  padding: 16px;
  border-radius: 4px;
  margin-bottom: 12px;
}
</style>
