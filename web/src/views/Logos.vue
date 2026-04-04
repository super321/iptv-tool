<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px">
      <div style="display: flex; align-items: center; gap: 12px">
        <h3 style="margin: 0">{{ $t('logos.title') }}</h3>
        <span style="font-size: 13px; color: var(--el-text-color-secondary)">
          {{ $t('logos.total_count', { count: filteredLogos.length }) }}
          {{ searchQuery ? $t('logos.filtered') : '' }}
        </span>
      </div>
      <div style="display: flex; align-items: center; gap: 12px">
        <el-input v-model="searchQuery" :placeholder="$t('logos.search_placeholder')" style="width: 220px" clearable :prefix-icon="Search" />
        <el-button type="danger" plain :icon="Delete" :disabled="!selectedRows.length"
                   @click="handleBatchDelete">
          {{ $t('logos.batch_delete') }}
          <span v-if="selectedRows.length"> ({{ selectedRows.length }})</span>
        </el-button>
        <el-upload :auto-upload="false" :show-file-list="false" accept="image/*" multiple
                   :on-change="onFileChange">
          <el-button type="primary" :loading="uploading">{{ $t('logos.upload') }}</el-button>
        </el-upload>
      </div>
    </div>

    <el-table :data="filteredLogos" v-loading="loading" border stripe
              @selection-change="onSelectionChange">
      <el-table-column type="selection" width="42" />
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column :label="$t('common.name')" min-width="180">
        <template #default="{ row }">
          <div v-if="row.isEditing" style="display: flex; gap: 8px; align-items: center">
            <el-input v-model="row.editName" size="small" @keyup.enter="saveName(row)" />
            <el-button size="small" type="success" :icon="Check" circle @click="saveName(row)" />
            <el-button size="small" type="info" :icon="Close" circle @click="cancelEdit(row)" />
          </div>
          <div v-else style="display: flex; align-items: center; justify-content: space-between;">
            <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" :title="row.name">{{ row.name }}</span>
            <el-button size="small" type="primary" link :icon="Edit" @click="startEdit(row)">{{ $t('common.edit') }}</el-button>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="$t('logos.col_preview')" width="100" align="center">
        <template #default="{ row }">
          <el-image
            :src="row.url_path"
            style="width: 40px; height: 40px; cursor: pointer"
            fit="contain"
            :preview-src-list="[row.url_path]"
            :z-index="3000"
            preview-teleported
            hide-on-click-modal
          />
        </template>
      </el-table-column>
      <el-table-column prop="url_path" :label="$t('logos.col_url_path')" min-width="250" show-overflow-tooltip />
      <el-table-column prop="created_at" :label="$t('logos.col_upload_time')" width="180">
        <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column :label="$t('common.operations')" width="100" fixed="right" align="center">
        <template #default="{ row }">
          <el-tooltip :content="$t('common.delete')" placement="top" :show-after="500">
            <el-button :icon="Delete" size="small" circle type="danger" @click="handleDelete(row)" />
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Check, Close, Search } from '@element-plus/icons-vue'
import api from '../api'

const { t } = useI18n()

const logos = ref([])
const loading = ref(false)
const uploading = ref(false)
const searchQuery = ref('')
const selectedRows = ref([])

const filteredLogos = computed(() => {
  if (!searchQuery.value) return logos.value
  const q = searchQuery.value.toLowerCase()
  return logos.value.filter(item => item.name && item.name.toLowerCase().includes(q))
})

// 收集批量选择的文件
let pendingFiles = []
let batchTimer = null

onMounted(() => loadLogos())

async function loadLogos() {
  loading.value = true
  try {
    const { data } = await api.get('/logos')
    // 初始化每个元素的编辑状态
    logos.value = (data || []).map(item => ({
      ...item,
      isEditing: false,
      editName: ''
    }))
  } finally { loading.value = false }
}

function onSelectionChange(rows) {
  selectedRows.value = rows
}

// el-upload on-change: 多选时每个文件触发一次，用 nextTick 合并为一次批量上传
function onFileChange(uploadFile) {
  pendingFiles.push(uploadFile.raw)
  if (batchTimer) clearTimeout(batchTimer)
  batchTimer = setTimeout(() => {
    const files = pendingFiles.slice()
    pendingFiles = []
    batchTimer = null
    doBatchUpload(files)
  }, 0)
}

async function doBatchUpload(files) {
  if (!files.length) return
  uploading.value = true
  try {
    const formData = new FormData()
    files.forEach(f => formData.append('files', f))
    const { data } = await api.post('/logos/batch-upload', formData)

    const successCount = (data.uploaded || []).length
    const failCount = (data.errors || []).length
    const total = successCount + failCount

    if (failCount === 0) {
      ElMessage.success(t('logos.batch_upload_success', { total }))
    } else if (successCount > 0) {
      // 部分失败：展示汇总 + 详细错误
      let msg = t('logos.batch_upload_partial', { total, success: successCount, fail: failCount })
      msg += '\n' + data.errors.join('\n')
      ElMessage.warning({ message: msg, duration: 5000 })
    } else {
      let msg = t('logos.batch_upload_all_failed', { total })
      msg += '\n' + data.errors.join('\n')
      ElMessage.error({ message: msg, duration: 5000 })
    }
    await loadLogos()
  } catch {
    ElMessage.error(t('logos.upload_failed'))
  } finally {
    uploading.value = false
  }
}

// 编辑台标名称逻辑
function startEdit(row) {
  row.editName = row.name
  row.isEditing = true
}

function cancelEdit(row) {
  row.isEditing = false
  row.editName = ''
}

async function saveName(row) {
  if (!row.editName || !row.editName.trim()) {
    ElMessage.warning(t('logos.name_empty'))
    return
  }
  
  const newName = row.editName.trim()
  
  if (newName === row.name) {
    cancelEdit(row)
    return
  }

  try {
    await api.put(`/logos/${row.id}`, { name: newName })
    row.name = newName
    ElMessage.success(t('logos.rename_success'))
    row.isEditing = false
  } catch (e) {
    // 错误在api拦截器已处理
  }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(t('logos.delete_confirm', { name: row.name }), t('common.confirm_delete'), { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') })
  await api.delete(`/logos/${row.id}`)
  ElMessage.success(t('common.delete_success'))
  await loadLogos()
}

async function handleBatchDelete() {
  if (!selectedRows.value.length) return
  const count = selectedRows.value.length
  await ElMessageBox.confirm(
    t('logos.batch_delete_confirm', { count }),
    t('common.confirm_delete'),
    { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') }
  )
  const ids = selectedRows.value.map(row => row.id)
  try {
    await api.post('/logos/batch-delete', { ids })
    ElMessage.success(t('logos.batch_delete_success', { count }))
    selectedRows.value = []
    await loadLogos()
  } catch {
    // 错误在api拦截器已处理
  }
}
</script>
