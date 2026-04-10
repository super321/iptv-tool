<template>
  <div>
    <div class="page-header">
      <div class="page-header-left">
        <h3>{{ $t('rules.title') }}</h3>
        <span class="text-secondary">
          {{ $t('rules.total_count', { count: filteredRulesList.length }) }}
          {{ searchQuery ? $t('common.filtered') : '' }}
        </span>
      </div>
      <div class="page-header-right">
        <el-input v-model="searchQuery" :placeholder="$t('rules.search_placeholder')" style="width: 220px" clearable :prefix-icon="Search" />
        <el-button type="primary" @click="showCreate">{{ $t('rules.add') }}</el-button>
      </div>
    </div>

    <el-table :data="filteredRulesList" v-loading="loading" border stripe>
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
            <el-button :icon="Edit" size="small" circle type="primary" @click="showEdit(row)" />
          </el-tooltip>
          <el-tooltip :content="$t('common.delete')" placement="top" :show-after="500">
            <el-button :icon="Delete" size="small" circle type="danger" @click="handleDelete(row)" />
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create/Edit Dialog -->
    <el-dialog v-if="dialogVisible" v-model="dialogVisible" :title="isEdit ? $t('rules.edit_title') : $t('rules.add_title')" width="700px" :close-on-click-modal="false">
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
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
            <span class="text-secondary">{{ $t('rules.alias_help') }}</span>
            <el-button size="small" type="success" @click="startAIGenerate('alias')" :icon="MagicStick">
              {{ $t('rules.ai_generate') }}
            </el-button>
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
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
            <span class="text-secondary">{{ $t('rules.filter_help') }}</span>
            <el-button size="small" type="success" @click="startAIGenerate('filter')" :icon="MagicStick">
              {{ $t('rules.ai_generate') }}
            </el-button>
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
                  <el-option :label="$t('rules.target_group')" value="group" />
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
            <span class="text-secondary">{{ $t('rules.group_help') }}</span>
            <el-button size="small" type="success" @click="startAIGenerate('group')" :icon="MagicStick">
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
              <el-button size="small" :icon="Delete" circle @click="group.rules.splice(rIdx, 1)" />
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
        <div style="display: flex; justify-content: space-between; width: 100%;">
          <el-button type="success" plain @click="openTestSourceDialog" :icon="Search" :disabled="!hasValidConfig">
            {{ $t('rules.test_btn') }}
          </el-button>
          <div>
            <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('common.confirm') }}</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- Test Rule: Source Selection Dialog -->
    <el-dialog v-if="testSourceDialogVisible" v-model="testSourceDialogVisible" :title="$t('rules.test_select_sources')" width="550px" destroy-on-close :close-on-click-modal="false">
      <p class="text-secondary" style="margin-top: 0">{{ $t('rules.test_select_sources_desc') }}</p>

      <!-- Source type switch -->
      <div style="margin-bottom: 16px">
        <div style="font-size: 14px; color: var(--el-text-color-regular); margin-bottom: 8px; font-weight: 500;">{{ $t('rules.test_source_type') }}</div>
        <el-radio-group v-model="testSourceType" :disabled="form.type === 'group'" @change="onTestSourceTypeChange">
          <el-radio value="live">{{ $t('rules.test_source_live') }}</el-radio>
          <el-radio value="epg">{{ $t('rules.test_source_epg') }}</el-radio>
        </el-radio-group>
        <div v-if="form.type === 'group'" class="help-text" style="margin-top: 6px">
          {{ $t('rules.test_source_type_locked') }}
        </div>
      </div>

      <div v-if="testSourcesLoading" v-loading="true" style="height: 120px"></div>
      <template v-else>
        <el-empty v-if="testSources.length === 0" :description="$t('rules.test_no_sources')" />
        <el-checkbox-group v-model="testSelectedSourceIds" v-else>
          <div v-for="src in testSources" :key="src.id" style="margin-bottom: 8px;">
            <el-checkbox :value="src.id">
              {{ src.name }}
              <el-tag size="small" type="info" style="margin-left: 6px;">{{ src.channel_count || src.epg_channel_count || 0 }}</el-tag>
            </el-checkbox>
          </div>
        </el-checkbox-group>
      </template>
      <template #footer>
        <el-button @click="testSourceDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="executeTest" :loading="testRunning" :disabled="testSelectedSourceIds.length === 0">
          {{ $t('rules.test_start') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Test Rule: Result Diff Dialog -->
    <el-dialog v-if="testResultDialogVisible" v-model="testResultDialogVisible" :title="$t('rules.test_result_title')" width="90vw" class="test-result-dialog" destroy-on-close :close-on-click-modal="false">
      <!-- Summary bar -->
      <div class="test-summary-bar">
        <div class="test-summary-item">
          <span class="test-summary-label">{{ $t('rules.test_summary_total') }}</span>
          <span class="test-summary-value">{{ testResult.summary.total }}</span>
        </div>
        <div class="test-summary-item test-summary-modified">
          <span class="test-summary-label">{{ $t('rules.test_summary_modified') }}</span>
          <span class="test-summary-value">{{ testResult.summary.modified }}</span>
        </div>
        <div class="test-summary-item test-summary-filtered">
          <span class="test-summary-label">{{ $t('rules.test_summary_filtered') }}</span>
          <span class="test-summary-value">{{ testResult.summary.filtered }}</span>
        </div>
        <div class="test-summary-item test-summary-unchanged">
          <span class="test-summary-label">{{ $t('rules.test_summary_unchanged') }}</span>
          <span class="test-summary-value">{{ testResult.summary.unchanged }}</span>
        </div>
      </div>

      <!-- Filter tabs + search -->
      <div class="test-toolbar">
        <el-radio-group v-model="testFilterMode" size="small">
          <el-radio-button value="all">{{ $t('rules.test_filter_all') }}</el-radio-button>
          <el-radio-button value="modified">{{ $t('rules.test_filter_modified') }}</el-radio-button>
          <el-radio-button value="filtered">{{ $t('rules.test_filter_filtered') }}</el-radio-button>
        </el-radio-group>
        <el-input v-model="testSearchQuery" :placeholder="$t('rules.test_search_placeholder')" style="width: 220px" clearable :prefix-icon="Search" size="small" />
      </div>

      <!-- Diff tables -->
      <div class="test-diff-container">
        <!-- Left: Original -->
        <div class="test-diff-panel">
          <div class="test-diff-header">{{ $t('rules.test_original') }}</div>
          <div class="test-diff-body">
            <table class="test-diff-table">
              <thead>
                <tr>
                  <th style="width: 50px">#</th>
                  <th>{{ $t('rules.test_col_name') }}</th>
                  <th v-if="testSourceType === 'live'">{{ $t('rules.test_col_group') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(item, idx) in filteredTestResults" :key="'o-'+idx" :class="testRowClass(testResult.applied[item._origIdx])">
                  <td class="test-row-num">{{ item._origIdx + 1 }}</td>
                  <td :class="{ 'test-text-strike': testResult.applied[item._origIdx]?.status === 'filtered' }">{{ item.name }}</td>
                  <td v-if="testSourceType === 'live'">{{ item.group || '-' }}</td>
                </tr>
                <tr v-if="filteredTestResults.length === 0">
                  <td :colspan="testSourceType === 'live' ? 3 : 2" style="text-align: center; color: var(--el-text-color-secondary); padding: 32px;">{{ $t('rules.test_no_data') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Divider -->
        <div class="test-diff-divider"></div>

        <!-- Right: Applied -->
        <div class="test-diff-panel">
          <div class="test-diff-header">{{ $t('rules.test_applied') }}</div>
          <div class="test-diff-body">
            <table class="test-diff-table">
              <thead>
                <tr>
                  <th style="width: 50px">#</th>
                  <th>{{ $t('rules.test_col_name') }}</th>
                  <th v-if="form.type === 'alias'">{{ $t('rules.test_col_alias') }}</th>
                  <th v-if="form.type === 'group'">{{ $t('rules.test_col_group') }}</th>
                  <th style="width: 80px">{{ $t('rules.test_col_status') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(item, idx) in filteredTestApplied" :key="'a-'+idx" :class="testRowClass(item)">
                  <td class="test-row-num">{{ item._origIdx + 1 }}</td>
                  <td>{{ item.alias || item.name }}</td>
                  <td v-if="form.type === 'alias'">
                    <span v-if="item.alias" class="test-alias-highlight">{{ item.alias }}</span>
                    <span v-else style="color: var(--el-text-color-placeholder)">-</span>
                  </td>
                  <td v-if="form.type === 'group'">{{ item.group || '-' }}</td>
                  <td>
                    <el-tag v-if="item.status === 'modified'" type="success" size="small">{{ $t('rules.test_summary_modified') }}</el-tag>
                    <el-tag v-else-if="item.status === 'filtered'" type="danger" size="small">{{ $t('rules.test_summary_filtered') }}</el-tag>
                    <span v-else style="color: var(--el-text-color-placeholder); font-size: 12px">-</span>
                  </td>
                </tr>
                <tr v-if="filteredTestApplied.length === 0">
                  <td :colspan="form.type === 'alias' ? 4 : (form.type === 'group' ? 4 : 3)" style="text-align: center; color: var(--el-text-color-secondary); padding: 32px;">{{ $t('rules.test_no_data') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- AI Step 1: Source Selection Dialog -->
    <el-dialog v-model="aiSourceDialogVisible" :title="$t('rules.ai_select_sources')" width="550px" destroy-on-close :close-on-click-modal="false">
      <p class="text-secondary" style="margin-top: 0">{{ $t('rules.ai_select_sources_desc') }}</p>
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
        <el-button type="primary" @click="aiLoadChannelsAndNext" :loading="aiChannelsLoading" :disabled="aiSelectedSourceIds.length === 0">
          {{ $t('rules.ai_next') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- AI Step 2: Intent Input + Channel List Dialog -->
    <el-dialog v-model="aiIntentDialogVisible" :title="aiIntentTitle" width="800px" destroy-on-close :close-on-click-modal="false">
      <div style="display: flex; gap: 16px; height: 420px;">
        <!-- Left: Channel name list with search -->
        <div style="flex: 0 0 280px; display: flex; flex-direction: column; min-width: 0;">
          <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px;">
            <span class="text-secondary" style="font-weight: 500;">{{ $t('rules.ai_channel_list_title') }}</span>
            <el-tag type="success" size="small">{{ aiChannelNames.length }}</el-tag>
          </div>
          <el-input
            v-model="aiChannelSearch"
            :placeholder="$t('rules.ai_channel_search')"
            size="small"
            clearable
            :prefix-icon="Search"
            style="margin-bottom: 8px;"
          />
          <div class="ai-channel-list">
            <div
              v-for="name in filteredAiChannelNames"
              :key="name"
              class="ai-channel-item"
            >{{ name }}</div>
            <div v-if="filteredAiChannelNames.length === 0" style="padding: 12px; color: var(--el-text-color-placeholder); text-align: center; font-size: 13px;">
              {{ $t('rules.ai_no_channels') }}
            </div>
          </div>
        </div>
        <!-- Right: Preset tags + Intent textarea -->
        <div style="flex: 1; display: flex; flex-direction: column; min-width: 0;">
          <div style="margin-bottom: 8px;">
            <span class="text-secondary" style="display: block; margin-bottom: 6px;">
              {{ aiIntentDesc }}
            </span>
          </div>
          <div style="margin-bottom: 10px; display: flex; flex-wrap: wrap; gap: 6px;">
            <el-tag
              v-for="preset in currentPresetTags"
              :key="preset.key"
              class="ai-preset-tag"
              effect="plain"
              round
              @click="applyPresetTag(preset.value)"
            >{{ preset.label }}</el-tag>
          </div>
          <el-input
            v-model="aiUserIntent"
            type="textarea"
            :rows="10"
            :placeholder="aiIntentPlaceholder"
            style="flex: 1;"
          />
          <div v-if="aiRuleType !== 'filter'" class="help-text" style="margin-top: 6px;">
            {{ aiRuleType === 'alias' ? $t('rules.ai_alias_intent_optional') : $t('rules.ai_group_intent_optional') }}
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="aiIntentDialogVisible = false; aiSourceDialogVisible = true">{{ $t('rules.ai_prev') }}</el-button>
        <el-button type="primary" @click="aiBuildPromptWithIntent" :disabled="aiRuleType === 'filter' && !aiUserIntent.trim()">
          {{ $t('rules.ai_next') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- AI Step 3: Prompt Display Dialog -->
    <el-dialog v-model="aiPromptDialogVisible" :title="$t('rules.ai_prompt_ready')" width="700px" destroy-on-close :close-on-click-modal="false">
      <el-alert :title="$t('rules.ai_prompt_instruction')" type="info" :closable="false" show-icon style="margin-bottom: 12px;" />
      <el-tag type="success" size="small" style="margin-bottom: 8px;">{{ $t('rules.ai_channel_count', { count: aiChannelNames.length }) }}</el-tag>
      <el-input v-model="aiPromptText" type="textarea" :rows="14" readonly style="font-family: monospace; font-size: 12px;" />
      <template #footer>
        <el-button @click="aiGoBackFromPrompt">{{ $t('rules.ai_prev') }}</el-button>
        <el-button @click="aiCopyPrompt" :icon="DocumentCopy">{{ $t('rules.ai_copy_prompt') }}</el-button>
        <el-button type="primary" @click="aiGoToResponse">{{ $t('rules.ai_next') }}</el-button>
      </template>
    </el-dialog>

    <!-- AI Step 4: Response Input Dialog -->
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
import { Delete, Edit, MagicStick, DocumentCopy, Search, Aim } from '@element-plus/icons-vue'
import api from '../api'
import { getPromptTemplate, validateGroupRulesJSON, validateAliasRulesJSON, validateFilterRulesJSON } from '../utils/promptTemplates'

const { t } = useI18n()

const rules = ref([])
const loading = ref(false)
const searchQuery = ref('')

const filteredRulesList = computed(() => {
  if (!searchQuery.value) return rules.value
  const q = searchQuery.value.toLowerCase()
  return rules.value.filter(r => r.name && r.name.toLowerCase().includes(q))
})
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const submitting = ref(false)
const formRef = ref()

const typeNameMap = computed(() => ({ alias: t('rules.type_alias'), filter: t('rules.type_filter'), group: t('rules.type_group') }))
const typeTagMap = { alias: 'primary', filter: 'danger', group: 'warning' }

const form = reactive({ name: '', description: '', type: 'alias', status: true })
const formRules = computed(() => ({
  name: [{ required: true, message: t('rules.rule_name_placeholder'), trigger: 'blur' }],
}))

// Config states
const aliasConfig = ref([])
const filterConfig = ref([])
const groupConfig = ref([])

// --- AI Generate states (unified for all rule types) ---
const aiRuleType = ref('')           // 'alias' | 'filter' | 'group'
const aiSourceDialogVisible = ref(false)
const aiIntentDialogVisible = ref(false)
const aiPromptDialogVisible = ref(false)
const aiResponseDialogVisible = ref(false)
const aiSources = ref([])
const aiSourcesLoading = ref(false)
const aiSelectedSourceIds = ref([])
const aiChannelsLoading = ref(false)
const aiChannelNames = ref([])
const aiChannelSearch = ref('')
const aiUserIntent = ref('')
const aiPromptText = ref('')
const aiResponseText = ref('')

// Filtered channel names for search in Intent dialog
const filteredAiChannelNames = computed(() => {
  if (!aiChannelSearch.value) return aiChannelNames.value
  const q = aiChannelSearch.value.toLowerCase()
  return aiChannelNames.value.filter(name => name.toLowerCase().includes(q))
})

// Intent dialog dynamic text based on rule type
const aiIntentTitle = computed(() => {
  const map = {
    alias: t('rules.ai_intent_title_alias'),
    filter: t('rules.ai_intent_title_filter'),
    group: t('rules.ai_intent_title_group'),
  }
  return map[aiRuleType.value] || ''
})
const aiIntentDesc = computed(() => {
  const map = {
    alias: t('rules.ai_intent_desc_alias'),
    filter: t('rules.ai_intent_desc_filter'),
    group: t('rules.ai_intent_desc_group'),
  }
  return map[aiRuleType.value] || ''
})
const aiIntentPlaceholder = computed(() => {
  const map = {
    alias: t('rules.ai_intent_placeholder_alias'),
    filter: t('rules.ai_intent_placeholder_filter'),
    group: t('rules.ai_intent_placeholder_group'),
  }
  return map[aiRuleType.value] || ''
})

// Preset tags per rule type
const currentPresetTags = computed(() => {
  if (aiRuleType.value === 'alias') {
    return [
      { key: 'hd', label: t('rules.ai_preset_alias_hd'), value: t('rules.ai_preset_alias_hd') },
      { key: 'cctv', label: t('rules.ai_preset_alias_cctv'), value: t('rules.ai_preset_alias_cctv') },
      { key: 'number', label: t('rules.ai_preset_alias_number'), value: t('rules.ai_preset_alias_number') },
      { key: 'space', label: t('rules.ai_preset_alias_space'), value: t('rules.ai_preset_alias_space') },
    ]
  }
  if (aiRuleType.value === 'group') {
    return [
      { key: 'cctv_sat', label: t('rules.ai_preset_group_cctv_sat'), value: t('rules.ai_preset_group_cctv_sat') },
      { key: 'by_region', label: t('rules.ai_preset_group_by_region'), value: t('rules.ai_preset_group_by_region') },
      { key: 'by_genre', label: t('rules.ai_preset_group_by_genre'), value: t('rules.ai_preset_group_by_genre') },
    ]
  }
  if (aiRuleType.value === 'filter') {
    return [
      { key: 'shopping', label: t('rules.ai_preset_filter_shopping'), value: t('rules.ai_preset_filter_shopping') },
      { key: 'sd', label: t('rules.ai_preset_filter_sd'), value: t('rules.ai_preset_filter_sd') },
      { key: 'test', label: t('rules.ai_preset_filter_test'), value: t('rules.ai_preset_filter_test') },
      { key: 'kids', label: t('rules.ai_preset_filter_kids'), value: t('rules.ai_preset_filter_kids') },
      { key: 'radio', label: t('rules.ai_preset_filter_radio'), value: t('rules.ai_preset_filter_radio') },
    ]
  }
  return []
})

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
// AI Generate Workflow (Unified)
// ===========================

async function startAIGenerate(ruleType) {
  aiRuleType.value = ruleType

  // Reset AI state
  aiSelectedSourceIds.value = []
  aiChannelNames.value = []
  aiChannelSearch.value = ''
  aiUserIntent.value = ''
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

async function aiLoadChannelsAndNext() {
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

    aiSourceDialogVisible.value = false
    // All types go to intent input dialog
    aiIntentDialogVisible.value = true
  } catch {
    ElMessage.error(t('rules.ai_no_channels'))
  } finally {
    aiChannelsLoading.value = false
  }
}

function applyPresetTag(text) {
  if (aiUserIntent.value.trim()) {
    // Append with separator
    aiUserIntent.value = aiUserIntent.value.trim() + '，' + text
  } else {
    aiUserIntent.value = text
  }
}

async function aiBuildPromptWithIntent() {
  // Validate for filter
  if (aiRuleType.value === 'filter' && !aiUserIntent.value.trim()) {
    ElMessage.warning(t('rules.ai_filter_intent_empty'))
    return
  }

  const templateMap = { alias: 'alias_rules', filter: 'filter_rules', group: 'group_rules' }
  const template = getPromptTemplate(templateMap[aiRuleType.value])
  aiPromptText.value = template.build(aiChannelNames.value, aiUserIntent.value)

  aiIntentDialogVisible.value = false
  aiPromptDialogVisible.value = true

  // Try auto-copy
  const ok = await copyToClipboard(aiPromptText.value)
  if (ok) {
    ElMessage.success(t('rules.ai_prompt_copied'))
  } else {
    ElMessage.warning(t('rules.ai_prompt_copy_failed'))
  }
}

function aiGoBackFromPrompt() {
  aiPromptDialogVisible.value = false
  aiIntentDialogVisible.value = true
}

async function aiCopyPrompt() {
  const ok = await copyToClipboard(aiPromptText.value)
  if (ok) {
    ElMessage.success(t('rules.ai_prompt_copied'))
  } else {
    ElMessage.warning(t('rules.ai_prompt_copy_failed'))
  }
}

function aiGoToResponse() {
  aiPromptDialogVisible.value = false
  aiResponseDialogVisible.value = true
}

/**
 * Copy text to clipboard with fallback for HTTP (non-secure) contexts.
 * navigator.clipboard is undefined on HTTP, so we fall back to execCommand.
 * @returns {Promise<boolean>} true if copy succeeded
 */
async function copyToClipboard(text) {
  // Prefer Clipboard API (requires HTTPS or localhost)
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Fall through to fallback
    }
  }
  // Fallback: hidden textarea + execCommand
  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.cssText = 'position:fixed;left:-9999px;top:-9999px;opacity:0'
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    return ok
  } catch {
    return false
  }
}

async function aiParseResponse() {
  const text = aiResponseText.value.trim()
  if (!text) {
    ElMessage.warning(t('rules.ai_parse_empty'))
    return
  }

  if (aiRuleType.value === 'group') {
    const result = validateGroupRulesJSON(text)
    if (!result.valid) {
      ElMessage.error(t('rules.ai_parse_error'))
      return
    }
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

  } else if (aiRuleType.value === 'alias') {
    const result = validateAliasRulesJSON(text)
    if (!result.valid) {
      ElMessage.error(t('rules.ai_alias_parse_error'))
      return
    }
    aliasConfig.value = result.data.map(r => ({
      match_mode: r.match_mode || 'regex',
      pattern: r.pattern,
      replacement: r.replacement ?? ''
    }))
    aiResponseDialogVisible.value = false
    ElMessage.success(t('rules.ai_alias_parse_success', { count: result.data.length }))

  } else if (aiRuleType.value === 'filter') {
    const result = validateFilterRulesJSON(text)
    if (!result.valid) {
      ElMessage.error(t('rules.ai_filter_parse_error'))
      return
    }
    filterConfig.value = result.data.map(r => ({
      target: r.target || 'name',
      match_mode: r.match_mode || 'regex',
      pattern: r.pattern
    }))
    aiResponseDialogVisible.value = false
    ElMessage.success(t('rules.ai_filter_parse_success', { count: result.data.length }))
  }
}
// ===========================
// Test Rule Workflow
// ===========================
const testSourceDialogVisible = ref(false)
const testResultDialogVisible = ref(false)
const testSourceType = ref('live')
const testSources = ref([])
const testSourcesLoading = ref(false)
const testSelectedSourceIds = ref([])
const testRunning = ref(false)
const testResult = reactive({
  original: [],
  applied: [],
  summary: { total: 0, modified: 0, filtered: 0, unchanged: 0 }
})
const testFilterMode = ref('all')
const testSearchQuery = ref('')

// Check if the current config has at least one valid rule
const hasValidConfig = computed(() => {
  if (form.type === 'alias') return aliasConfig.value.some(r => r.pattern?.trim())
  if (form.type === 'filter') return filterConfig.value.some(r => r.pattern?.trim())
  if (form.type === 'group') return groupConfig.value.some(g => g.group_name?.trim() && g.rules?.some(r => r.pattern?.trim()))
  return false
})

// Build current config data (same logic as handleSubmit)
function buildCurrentConfig() {
  if (form.type === 'alias') {
    return aliasConfig.value.filter(r => r.pattern.trim())
  } else if (form.type === 'filter') {
    return filterConfig.value.filter(r => r.pattern.trim())
  } else if (form.type === 'group') {
    return groupConfig.value.filter(g => g.group_name.trim()).map(g => ({
      group_name: g.group_name,
      rules: g.rules.filter(r => r.pattern.trim())
    })).filter(g => g.rules.length > 0)
  }
  return []
}

async function openTestSourceDialog() {
  const config = buildCurrentConfig()
  if (config.length === 0) {
    ElMessage.warning(t('rules.test_config_empty'))
    return
  }

  // Reset test state
  testSelectedSourceIds.value = []
  testSources.value = []
  testFilterMode.value = 'all'
  testSearchQuery.value = ''

  // Group type forces live source (group rules only apply to live channels)
  if (form.type === 'group') {
    testSourceType.value = 'live'
  } else {
    testSourceType.value = 'live'
  }

  testSourceDialogVisible.value = true
  await loadTestSources()
}

async function loadTestSources() {
  testSourcesLoading.value = true
  testSources.value = []
  try {
    if (testSourceType.value === 'live') {
      const { data } = await api.get('/live-sources')
      testSources.value = (data || []).filter(s => s.channel_count > 0)
    } else {
      const { data } = await api.get('/epg-sources')
      // EPG sources: use channel_count to show availability
      testSources.value = (data || []).filter(s => (s.channel_count || 0) > 0).map(s => ({
        ...s,
        epg_channel_count: s.channel_count || 0
      }))
    }
  } catch {
    testSources.value = []
  } finally {
    testSourcesLoading.value = false
  }
}

function onTestSourceTypeChange() {
  testSelectedSourceIds.value = []
  loadTestSources()
}

async function executeTest() {
  const config = buildCurrentConfig()
  if (config.length === 0) {
    ElMessage.warning(t('rules.test_config_empty'))
    return
  }

  testRunning.value = true
  try {
    const { data } = await api.post('/rules/test', {
      type: form.type,
      config: config,
      source_type: testSourceType.value,
      source_ids: testSelectedSourceIds.value
    })

    testResult.original = data.original || []
    testResult.applied = data.applied || []
    testResult.summary = data.summary || { total: 0, modified: 0, filtered: 0, unchanged: 0 }

    // Add _origIdx for tracking
    testResult.original.forEach((item, idx) => item._origIdx = idx)
    testResult.applied.forEach((item, idx) => item._origIdx = idx)

    testSourceDialogVisible.value = false
    testResultDialogVisible.value = true
  } catch (err) {
    if (err.response?.data?.error) {
      ElMessage.error(err.response.data.error)
    }
  } finally {
    testRunning.value = false
  }
}

// Filtered results for diff view
const filteredTestResults = computed(() => {
  let items = testResult.original.map((item, idx) => ({ ...item, _origIdx: idx }))

  // Filter by status
  if (testFilterMode.value === 'modified') {
    items = items.filter((_, idx) => testResult.applied[idx]?.status === 'modified')
  } else if (testFilterMode.value === 'filtered') {
    items = items.filter((_, idx) => testResult.applied[idx]?.status === 'filtered')
  }

  // Filter by search
  if (testSearchQuery.value) {
    const q = testSearchQuery.value.toLowerCase()
    items = items.filter(item => {
      const applied = testResult.applied[item._origIdx]
      return item.name.toLowerCase().includes(q) || (applied?.alias || '').toLowerCase().includes(q)
    })
  }

  return items
})

const filteredTestApplied = computed(() => {
  const origIdxSet = new Set(filteredTestResults.value.map(i => i._origIdx))
  return testResult.applied
    .map((item, idx) => ({ ...item, _origIdx: idx }))
    .filter(item => origIdxSet.has(item._origIdx))
})

function testRowClass(appliedItem) {
  if (!appliedItem) return ''
  if (appliedItem.status === 'filtered') return 'test-row-filtered'
  if (appliedItem.status === 'modified') return 'test-row-modified'
  return ''
}
</script>

<style scoped>
.rule-box {
  border: 1px solid var(--el-border-color-lighter);
  padding: 16px;
  border-radius: 4px;
  margin-bottom: 12px;
}

.ai-channel-list {
  flex: 1;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 4px;
}

.ai-channel-item {
  padding: 4px 10px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  border-radius: 4px;
  line-height: 1.6;
  transition: background-color 0.15s;
}

.ai-channel-item:hover {
  background-color: var(--el-fill-color-light);
}

.ai-preset-tag {
  cursor: pointer;
  transition: all 0.2s;
}

.ai-preset-tag:hover {
  color: var(--el-color-success);
  border-color: var(--el-color-success);
  background-color: var(--el-color-success-light-9);
}

/* ===== Test Rule Diff Styles ===== */

.test-summary-bar {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.test-summary-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.test-summary-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.test-summary-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.test-summary-modified .test-summary-value {
  color: var(--el-color-success);
}

.test-summary-filtered .test-summary-value {
  color: var(--el-color-danger);
}

.test-summary-unchanged .test-summary-value {
  color: var(--el-text-color-secondary);
}

.test-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.test-diff-container {
  display: flex;
  gap: 0;
  height: 55vh;
  max-height: 600px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.test-diff-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.test-diff-header {
  padding: 10px 16px;
  font-weight: 600;
  font-size: 13px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}

.test-diff-body {
  flex: 1;
  overflow-y: auto;
  overflow-x: auto;
}

.test-diff-divider {
  width: 1px;
  background: var(--el-border-color-lighter);
  flex-shrink: 0;
}

.test-diff-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.test-diff-table th {
  padding: 8px 12px;
  text-align: left;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
  position: sticky;
  top: 0;
  z-index: 1;
  white-space: nowrap;
}

.test-diff-table td {
  padding: 6px 12px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  color: var(--el-text-color-regular);
}

.test-row-num {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.test-row-modified {
  background-color: var(--el-color-success-light-9);
}

.test-row-filtered {
  background-color: var(--el-color-danger-light-9);
}

.test-text-strike {
  text-decoration: line-through;
  color: var(--el-text-color-placeholder);
}

.test-alias-highlight {
  color: var(--el-color-success);
  font-weight: 500;
}
</style>

<style>
/* Override dialog max-width for test result dialog specifically */
.test-result-dialog {
  max-width: 1200px;
}
</style>
