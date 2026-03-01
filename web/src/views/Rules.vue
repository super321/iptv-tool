<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <h3 style="margin: 0">聚合规则</h3>
      <el-button type="primary" @click="showCreate">新增规则</el-button>
    </div>

    <el-table :data="rules" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="规则名称" width="150" show-overflow-tooltip />
      <el-table-column prop="description" label="描述" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="typeTagMap[row.type]" size="small">{{ typeNameMap[row.type] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="规则数" width="100" align="center">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ getRuleCount(row.config, row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'" size="small">{{ row.status ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right" align="center">
        <template #default="{ row }">
          <el-tooltip content="编辑" placement="top" :show-after="500">
            <el-button icon="Edit" size="small" circle type="primary" @click="showEdit(row)" />
          </el-tooltip>
          <el-tooltip content="删除" placement="top" :show-after="500">
            <el-button icon="Delete" size="small" circle type="danger" @click="handleDelete(row)" />
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create/Edit Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑规则' : '新增规则'" width="700px" destroy-on-close :close-on-click-modal="false">
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入规则名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" placeholder="可选的规则描述说明" />
        </el-form-item>
        <el-form-item label="规则类型" prop="type" v-if="!isEdit">
          <el-radio-group v-model="form.type" @change="onTypeChange">
            <el-radio value="alias">频道别名</el-radio>
            <el-radio value="filter">频道过滤</el-radio>
            <el-radio value="group">频道分组</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item label="状态" v-if="isEdit">
          <el-switch v-model="form.status" />
        </el-form-item>

        <!-- Dynamic Rule Configs based on Type -->
        <el-divider>配置详情</el-divider>
        
        <!-- Type: Alias -->
        <template v-if="form.type === 'alias'">
          <div style="margin-bottom: 8px; color: #909399; font-size: 13px;">
            设置频道别名。引擎会按顺序匹配，命中第一条即生效。正则模式下可使用 $1, $2 替换分组。
          </div>
          <div v-for="(rule, idx) in aliasConfig" :key="idx" class="rule-box">
            <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px;">
              <span style="font-weight: bold; color: #606266; font-size: 14px;">别名规则 {{ idx + 1 }}</span>
              <el-button size="small" type="danger" link @click="aliasConfig.splice(idx, 1)">删除</el-button>
            </div>
            <el-row :gutter="10">
              <el-col :span="6">
                <el-select v-model="rule.match_mode" size="small" placeholder="匹配模式">
                  <el-option label="正则表达式" value="regex" />
                  <el-option label="普通字符串" value="string" />
                </el-select>
              </el-col>
              <el-col :span="9">
                <el-input v-model="rule.pattern" size="small" placeholder="匹配内容 (如 ^CCTV-(.*)$)" />
              </el-col>
              <el-col :span="9">
                <el-input v-model="rule.replacement" size="small" placeholder="替换别名 (如 CCTV$1)" />
              </el-col>
            </el-row>
          </div>
          <el-button size="small" style="width: 100%" @click="aliasConfig.push({ match_mode: 'regex', pattern: '', replacement: '' })">
            + 添加别名规则
          </el-button>
        </template>

        <!-- Type: Filter -->
        <template v-if="form.type === 'filter'">
          <div style="margin-bottom: 8px; color: #909399; font-size: 13px;">
            过滤掉不需要的频道或节目。引擎会按顺序匹配，命中即丢弃。
          </div>
          <div v-for="(rule, idx) in filterConfig" :key="idx" class="rule-box">
            <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px;">
              <span style="font-weight: bold; color: #606266; font-size: 14px;">过滤规则 {{ idx + 1 }}</span>
              <el-button size="small" type="danger" link @click="filterConfig.splice(idx, 1)">删除</el-button>
            </div>
            <el-row :gutter="10">
              <el-col :span="6">
                <el-select v-model="rule.target" size="small" placeholder="匹配目标">
                  <el-option label="原始名称" value="name" />
                  <el-option label="频道别名" value="alias" />
                </el-select>
              </el-col>
              <el-col :span="6">
                <el-select v-model="rule.match_mode" size="small" placeholder="匹配模式">
                  <el-option label="正则表达式" value="regex" />
                  <el-option label="普通字符串" value="string" />
                </el-select>
              </el-col>
              <el-col :span="12">
                <el-input v-model="rule.pattern" size="small" placeholder="匹配内容 (如 .*购物.*)" />
              </el-col>
            </el-row>
          </div>
          <el-button size="small" style="width: 100%" @click="filterConfig.push({ target: 'name', match_mode: 'regex', pattern: '' })">
            + 添加过滤规则
          </el-button>
        </template>

        <!-- Type: Group -->
        <template v-if="form.type === 'group'">
          <div style="margin-bottom: 8px; color: #909399; font-size: 13px;">
            频道分组（仅直播生效）。从上到下匹配，命中后归入指定分组。
          </div>
          <div v-for="(group, gIdx) in groupConfig" :key="gIdx" class="rule-box" style="background: #fafafa">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px">
              <el-input v-model="group.group_name" size="small" placeholder="分组名称 (如：央视)" style="width: 200px" />
              <el-button size="small" type="danger" link @click="groupConfig.splice(gIdx, 1)">删除分组</el-button>
            </div>
            
            <div v-for="(rule, rIdx) in group.rules" :key="rIdx" style="margin-bottom: 6px; display: flex; gap: 8px">
              <el-select v-model="rule.target" size="small" style="width: 100px" placeholder="匹配目标">
                <el-option label="原始名称" value="name" />
                <el-option label="频道别名" value="alias" />
              </el-select>
              <el-select v-model="rule.match_mode" size="small" style="width: 110px">
                <el-option label="正则表达式" value="regex" />
                <el-option label="普通字符串" value="string" />
              </el-select>
              <el-input v-model="rule.pattern" size="small" placeholder="匹配内容" style="flex: 1" />
              <el-button size="small" icon="Delete" circle @click="group.rules.splice(rIdx, 1)" />
            </div>
            <el-button size="small" type="primary" plain @click="group.rules.push({ target: 'name', match_mode: 'regex', pattern: '' })">
              + 增加匹配条件
            </el-button>
          </div>
          <el-button size="small" style="width: 100%" @click="groupConfig.push({ group_name: '', rules: [{ target: 'name', match_mode: 'regex', pattern: '' }] })">
            + 添加分组
          </el-button>
        </template>

      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit } from '@element-plus/icons-vue'
import api from '../api'

const rules = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const submitting = ref(false)
const formRef = ref()

const typeNameMap = { alias: '频道别名', filter: '频道过滤', group: '频道分组' }
const typeTagMap = { alias: 'primary', filter: 'danger', group: 'warning' }

const form = reactive({ name: '', type: 'alias', status: true })
const formRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
}

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
    ElMessage.warning('规则内容不能为空，请至少填写一条有效规则')
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
      ElMessage.success('更新成功')
    } else {
      await api.post('/rules', body)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await loadRules()
  } catch {}
  finally { submitting.value = false }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(`确定删除规则 "${row.name}"？`, '确认删除', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' })
  await api.delete(`/rules/${row.id}`)
  ElMessage.success('删除成功')
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