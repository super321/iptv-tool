<template>
  <div>
    <div class="page-header">
      <div class="page-header-left">
        <h3>{{ $t('live_sources.title') }}</h3>
        <span class="text-secondary">
          {{ $t('live_sources.total_count', { count: filteredSources.length }) }}
          {{ searchQuery ? $t('common.filtered') : '' }}
        </span>
      </div>
      <div class="page-header-right">
        <el-input v-model="searchQuery" :placeholder="$t('live_sources.search_placeholder')" style="width: 220px" clearable :prefix-icon="Search" />
        <el-button type="primary" @click="showCreate">{{ $t('live_sources.add') }}</el-button>
      </div>
    </div>

    <el-table :data="filteredSources" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" :label="$t('common.name')" min-width="140" show-overflow-tooltip />
      <el-table-column prop="description" :label="$t('common.description')" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.description || '-' }}</template>
      </el-table-column>
      <el-table-column prop="type" :label="$t('common.type')" width="140">
        <template #default="{ row }">
          <el-tag :type="typeTagMap[row.type]" size="small">{{ typeNameMap[row.type] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="cron_time" :label="$t('live_sources.scheduled_refresh')" width="140">
        <template #default="{ row }">{{ formatSchedule(row.cron_time) }}</template>
      </el-table-column>
      <el-table-column prop="status" :label="$t('common.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'" size="small">{{ row.status ? $t('common.enabled') : $t('common.disabled') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="channel_count" :label="$t('live_sources.channel_count')" width="120" align="center">
        <template #default="{ row }">
          <el-tag type="info" size="small">{{ row.channel_count || 0 }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.update_time')" width="200">
        <template #default="{ row }">
          <div v-if="row.is_syncing" style="display: flex; align-items: center; gap: 6px; color: var(--el-color-primary)">
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
          <el-tooltip :content="$t('live_sources.tooltip_channels')" placement="top" :show-after="500">
            <el-button :icon="List" size="small" circle @click="showChannels(row)" />
          </el-tooltip>
          <el-tooltip :content="$t('common.tooltip_sync')" placement="top" :show-after="500">
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
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('live_sources.edit_title') : $t('live_sources.add_title')" width="680px" destroy-on-close :close-on-click-modal="false">
      <el-form :model="form" :rules="formRules" ref="formRef" label-width="120px">
        <el-form-item :label="$t('common.name')" prop="name">
          <el-input v-model.trim="form.name" />
        </el-form-item>
        <el-form-item :label="$t('common.description')">
          <el-input v-model.trim="form.description" :placeholder="$t('common.optional_description')" />
        </el-form-item>
        <el-form-item :label="$t('common.type')" prop="type" v-if="!isEdit">
          <el-select v-model="form.type" style="width: 100%" @change="onTypeChange">
            <el-option :label="$t('live_sources.type_iptv_stb')" value="iptv" />
            <el-option :label="$t('live_sources.type_network_url_label')" value="network_url" />
            <el-option :label="$t('live_sources.type_network_manual')" value="network_manual" />
          </el-select>
        </el-form-item>

        <!-- network_url fields -->
        <el-form-item :label="$t('live_sources.subscribe_url')" v-if="form.type === 'network_url'" prop="url">
          <el-input v-model.trim="form.url" :placeholder="$t('live_sources.subscribe_url_placeholder')" />
        </el-form-item>

        <template v-if="form.type === 'network_url'">
          <el-divider content-position="left">{{ $t('common.custom_headers') }}</el-divider>
          <div style="margin-bottom: 16px; padding: 0 40px">
            <div v-for="(header, idx) in form.network_headers" :key="idx" style="display: flex; gap: 8px; margin-bottom: 8px">
              <el-input v-model="header.name" :placeholder="$t('common.header_name')" style="width: 200px" />
              <el-input v-model="header.value" :placeholder="$t('common.header_value')" style="flex: 1" />
              <el-button :icon="Delete" circle size="small" @click="form.network_headers.splice(idx, 1)" />
            </div>
            <el-button size="small" @click="form.network_headers.push({ name: '', value: '' })">
              <el-icon><Plus /></el-icon> {{ $t('common.add_header') }}
            </el-button>
          </div>
        </template>

        <!-- network_manual fields -->
        <el-form-item :label="$t('live_sources.content')" v-if="form.type === 'network_manual'" prop="content">
          <el-input v-model="form.content" type="textarea" :rows="8" :placeholder="$t('live_sources.content_placeholder')" />
        </el-form-item>

        <!-- === IPTV specific fields === -->
        <template v-if="form.type === 'iptv'">
          <el-divider content-position="left">{{ $t('live_sources.iptv_config') }}</el-divider>

          <el-form-item :label="$t('live_sources.iptv_platform')" prop="iptv.platform">
            <el-select v-model="form.iptv.platform" style="width: 100%">
              <el-option :label="$t('live_sources.huawei')" value="huawei" />
            </el-select>
          </el-form-item>

          <el-form-item :label="$t('live_sources.server_host')" prop="iptv.serverHost">
            <el-input v-model.trim="form.iptv.serverHost" :placeholder="$t('live_sources.server_host_placeholder')" />
          </el-form-item>

          <el-form-item :label="$t('live_sources.provider')" prop="iptv.providerSuffix">
            <el-select v-model="form.iptv.providerSuffix" style="width: 100%">
              <el-option :label="$t('live_sources.ctc')" value="CTC" />
              <el-option :label="$t('live_sources.cu')" value="CU" />
            </el-select>
          </el-form-item>

          <el-form-item :label="$t('live_sources.encrypt_key')" prop="iptv.key">
            <div style="display: flex; gap: 8px; width: 100%">
              <el-input v-model.trim="form.iptv.key" :placeholder="$t('live_sources.key_placeholder')" style="flex: 1" />
              <el-button type="success" @click="showCrackDialog">{{ $t('live_sources.crack_key') }}</el-button>
            </div>
          </el-form-item>

          <el-form-item :label="$t('live_sources.client_ip')" prop="iptv.ip">
            <el-input v-model.trim="form.iptv.ip" :placeholder="$t('live_sources.client_ip_placeholder')" />
            <div class="help-text">
              {{ $t('live_sources.client_ip_help') }}
            </div>
          </el-form-item>

          <el-divider content-position="left">{{ $t('common.custom_headers') }}</el-divider>
          <div style="margin-bottom: 16px">
            <div v-for="(header, idx) in form.iptv.headersList" :key="idx" style="display: flex; gap: 8px; margin-bottom: 8px">
              <el-input v-model="header.name" :placeholder="$t('common.header_name')" style="width: 200px" />
              <el-input v-model="header.value" :placeholder="$t('common.header_value')" style="flex: 1" />
              <el-button :icon="Delete" circle size="small" @click="form.iptv.headersList.splice(idx, 1)" />
            </div>
            <el-button size="small" @click="form.iptv.headersList.push({ name: '', value: '' })">
              <el-icon><Plus /></el-icon> {{ $t('common.add_header') }}
            </el-button>
          </div>

          <el-divider content-position="left">{{ $t('live_sources.auth_params_section') }}</el-divider>
          <el-form-item :label="$t('live_sources.auth_params')" prop="iptv.authParamsStr">
            <el-input v-model="form.iptv.authParamsStr" type="textarea" :rows="10"
              :placeholder="$t('live_sources.auth_params_placeholder')" />
            <div class="help-text">
              {{ $t('live_sources.auth_params_help') }}
              <el-link type="primary" :underline="false" @click="fillAuthExample" style="font-size: 12px">{{ $t('live_sources.fill_example') }}</el-link>
            </div>
          </el-form-item>
        </template>

        <!-- Scheduled Refresh -->
        <el-form-item :label="$t('live_sources.scheduled_refresh')" v-if="form.type !== 'network_manual'">
          <ScheduleConfig v-model="form.cron_time" i18n-prefix="live_sources" :enable-label="$t('live_sources.scheduled_refresh')" />
        </el-form-item>

        <!-- Scheduled Detect -->
        <el-form-item :label="$t('live_sources.scheduled_detect')">
          <ScheduleConfig v-model="form.cron_detect" i18n-prefix="live_sources" :enable-label="$t('live_sources.scheduled_detect')" />
          <div class="help-text">
            {{ $t('live_sources.scheduled_detect_help') }}
          </div>
        </el-form-item>

        <!-- Detect Strategy (shown when cron_detect is set) -->
        <el-form-item :label="$t('live_sources.detect_strategy')" v-if="form.cron_detect">
          <el-select v-model="form.detect_strategy" style="width: 100%">
            <el-option :label="$t('live_sources.detect_strategy_unicast')" value="unicast" />
            <el-option :label="$t('live_sources.detect_strategy_multicast')" value="multicast" />
          </el-select>
          <div class="help-text">
            {{ $t('live_sources.detect_strategy_help') }}
          </div>
        </el-form-item>

        <!-- EPG sync for IPTV -->
        <template v-if="form.type === 'iptv' && !isEdit">
          <el-form-item :label="$t('live_sources.sync_epg')">
            <el-switch v-model="form.epg_enabled" />
            <span class="form-hint" style="margin-left: 8px">{{ $t('live_sources.auto_create_epg') }}</span>
          </el-form-item>
          <el-form-item :label="$t('live_sources.epg_strategy')" v-if="form.epg_enabled">
            <el-select v-model="form.iptv.epgStrategy" style="width: 100%">
              <el-option v-for="opt in epgStrategies" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <div class="help-text">
              {{ $t('live_sources.epg_strategy_help') }}
            </div>
          </el-form-item>
        </template>

        <!-- Auto-create EPG for network types -->
        <template v-if="(form.type === 'network_url' || form.type === 'network_manual')">
          <el-form-item :label="$t('live_sources.auto_create_epg_label')">
            <el-switch v-model="form.epg_enabled" />
            <div class="help-text">
              {{ $t('live_sources.auto_create_epg_help') }}
            </div>
          </el-form-item>
        </template>

        <!-- Status (edit only) -->
        <el-form-item :label="$t('common.status')" v-if="isEdit">
          <el-switch v-model="form.status" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- Crack Key Dialog -->
    <el-dialog v-model="crackDialogVisible" :title="$t('live_sources.crack_title')" width="600px" destroy-on-close :close-on-click-modal="false" @close="onCrackDialogClose">
      <el-form label-width="110px">
        <el-form-item label="Authenticator">
          <el-input v-model="crackAuthenticator" type="textarea" :rows="3"
            :placeholder="$t('live_sources.crack_auth_placeholder')" :disabled="cracking" />
        </el-form-item>
        <el-form-item :label="$t('live_sources.crack_mode')">
          <el-radio-group v-model="crackMode" :disabled="cracking">
            <el-radio value="decimal">
              {{ $t('live_sources.crack_mode_decimal') }}
              <span style="color: var(--el-text-color-secondary); font-size: 12px; margin-left: 4px">
                ({{ $t('live_sources.crack_decimal_total') }})
              </span>
            </el-radio>
            <el-radio value="hex">
              {{ $t('live_sources.crack_mode_hex') }}
            </el-radio>
          </el-radio-group>
          <el-alert v-if="crackMode === 'hex'" :title="$t('live_sources.crack_hex_warning')" type="warning" show-icon :closable="false" style="margin-top: 8px" />
        </el-form-item>

        <!-- Progress -->
        <template v-if="cracking || crackProgress">
          <el-form-item :label="$t('live_sources.cracking')">
            <div style="width: 100%">
              <el-progress :percentage="crackProgress ? Math.min(100, crackProgress.percent) : 0" :stroke-width="18" :text-inside="true"
                :format="() => crackProgress ? Math.min(100, crackProgress.percent).toFixed(2) + '%' : '0%'" />
              <div v-if="crackProgress" style="margin-top: 6px; font-size: 12px; color: var(--el-text-color-secondary)">
                {{ $t('live_sources.crack_progress_detail', { tried: formatNumber(crackProgress.tried), total: formatNumber(crackProgress.total) }) }}
              </div>
            </div>
          </el-form-item>
        </template>

        <!-- Success result -->
        <template v-if="crackResult">
          <el-alert :title="$t('live_sources.crack_success')" type="success" show-icon :closable="false" style="margin-bottom: 16px">
            <template #default>
              <span style="font-size: 16px; font-weight: 700; letter-spacing: 1px; font-family: 'Courier New', monospace">
                {{ $t('live_sources.crack_key_label') }}: {{ crackResult.key }}
              </span>
            </template>
          </el-alert>
          <el-descriptions :title="$t('live_sources.crack_fields_title')" :column="1" border size="small" style="margin-bottom: 12px">
            <el-descriptions-item v-for="field in crackResult.fields" :key="field.name" :label="field.name" label-class-name="crack-field-label">
              <span style="font-family: 'Courier New', monospace; word-break: break-all; user-select: text">{{ field.value || '-' }}</span>
            </el-descriptions-item>
          </el-descriptions>
        </template>

        <!-- Error / Stopped -->
        <el-alert v-if="crackError" :title="crackError" type="error" show-icon :closable="false" style="margin-bottom: 12px" />
        <el-alert v-if="crackStopped" :title="$t('live_sources.crack_stopped')" type="info" show-icon :closable="false" style="margin-bottom: 12px" />
      </el-form>
      <template #footer>
        <el-button @click="crackDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button v-if="cracking" type="danger" @click="stopCrack">{{ $t('live_sources.crack_stop') }}</el-button>
        <el-button v-else type="primary" @click="doCrack">
          {{ $t('live_sources.crack_start') }}
        </el-button>
        <el-button v-if="crackResult" type="success" @click="applyCrackResult">{{ $t('live_sources.crack_apply') }}</el-button>
      </template>
    </el-dialog>


    <!-- Channels Dialog -->
    <el-dialog v-model="channelsVisible" :title="$t('live_sources.channels_title')" width="1100px" destroy-on-close :close-on-click-modal="false">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
        <span class="text-secondary">
          {{ $t('common.channels_total', { count: filteredChannels.length }) }} {{ channelsSearch ? $t('common.filtered') : '' }}
          <span v-if="channelsDetecting" style="margin-left: 12px; color: var(--el-color-warning)">
            {{ $t('live_sources.detect_progress', { detected: detectedCount, total: channels.length }) }}
          </span>
        </span>
        <div style="display: flex; gap: 12px; align-items: center">
          <el-input v-model="channelsSearch" :placeholder="$t('common.search_channel')" style="width: 200px" size="small" clearable @input="handleSearchChange" />
          <el-select v-model="detectStrategy" size="small" style="width: 140px">
            <el-option :label="$t('live_sources.detect_strategy_unicast')" value="unicast" />
            <el-option :label="$t('live_sources.detect_strategy_multicast')" value="multicast" />
          </el-select>
          <el-button type="warning" size="small" @click="triggerDetect" :loading="detectTriggering" :disabled="channelsDetecting">
            {{ channelsDetecting ? $t('live_sources.detecting') : $t('live_sources.detect_btn') }}
          </el-button>
        </div>
      </div>
      <el-table :data="paginatedChannels" max-height="400" border stripe size="small" style="user-select: text">
        <el-table-column prop="tvg_id" :label="$t('common.col_channel_id')" width="150" show-overflow-tooltip />
        <el-table-column prop="name" :label="$t('live_sources.col_channel_name')" min-width="150" />
        <el-table-column prop="group" :label="$t('live_sources.col_group')" min-width="120" />
        <el-table-column :label="$t('live_sources.col_latency')" width="100" align="center">
          <template #default="{ row }">
            <span v-if="channelsDetecting && (row.latency === null || row.latency === undefined)">
              <el-icon class="is-loading" :size="14"><Loading /></el-icon>
            </span>
            <span v-else-if="row.latency === null || row.latency === undefined">-</span>
            <span v-else-if="row.latency === -1" style="color: var(--el-color-danger); font-weight: 600">Timeout</span>
            <span v-else-if="row.latency < 500" style="color: var(--el-color-success); font-weight: 600">{{ row.latency }}ms</span>
            <span v-else-if="row.latency < 1000" style="color: var(--el-color-primary); font-weight: 600">{{ row.latency }}ms</span>
            <span v-else style="color: var(--el-color-warning); font-weight: 600">{{ row.latency }}ms</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('live_sources.col_video_codec')" width="120" align="center">
          <template #default="{ row }">
            <span v-if="channelsDetecting && (row.video_codec === null || row.video_codec === undefined) && (row.latency === null || row.latency === undefined)">
              <el-icon class="is-loading" :size="14"><Loading /></el-icon>
            </span>
            <span v-else-if="row.video_codec" style="font-weight: 600">{{ row.video_codec }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('live_sources.col_video_resolution')" width="140" align="center">
          <template #default="{ row }">
            <span v-if="channelsDetecting && (row.video_resolution === null || row.video_resolution === undefined) && (row.latency === null || row.latency === undefined)">
              <el-icon class="is-loading" :size="14"><Loading /></el-icon>
            </span>
            <span v-else-if="row.video_resolution" style="font-weight: 600">{{ row.video_resolution }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('live_sources.col_address')" min-width="200" align="center">
          <template #default="{ row }">
            <el-popover trigger="click" width="520" placement="left" :show-arrow="true">
              <template #reference>
                <div style="cursor: pointer; display: flex; flex-direction: column; gap: 4px; align-items: center">
                  <el-tag size="small" type="primary">
                    {{ getUrls(row).length }} {{ $t('live_sources.live_addr_unit') }}
                  </el-tag>
                  <el-tag v-if="row.catchup_url" size="small" type="warning">
                    {{ $t('live_sources.has_catchup') }}
                  </el-tag>
                </div>
              </template>
              <div style="user-select: text">
                <div style="font-weight: 600; margin-bottom: 8px; font-size: 13px; color: var(--el-text-color-primary)">{{ $t('live_sources.live_addresses') }}</div>
                <div v-for="(u, i) in getUrls(row)" :key="i" style="margin-bottom: 6px; display: flex; align-items: flex-start; gap: 6px">
                  <el-tag size="small" :type="u.startsWith('igmp://') || u.startsWith('rtp://') ? 'warning' : 'success'" style="flex-shrink: 0; margin-top: 1px">
                    {{ u.startsWith('igmp://') || u.startsWith('rtp://') ? $t('live_sources.multicast') : $t('live_sources.unicast') }}
                  </el-tag>
                  <span style="font-size: 12px; word-break: break-all; color: var(--el-text-color-regular); line-height: 1.5">{{ u }}</span>
                </div>
                <template v-if="row.catchup_url">
                  <el-divider style="margin: 10px 0" />
                  <div style="font-weight: 600; margin-bottom: 8px; font-size: 13px; color: var(--el-text-color-primary)">
                    {{ $t('live_sources.catchup_address') }}
                    <el-tag v-if="row.catchup_days" size="small" type="info" style="margin-left: 8px">
                      {{ $t('live_sources.catchup_days_label', { days: row.catchup_days }) }}
                    </el-tag>
                  </div>
                  <span style="font-size: 12px; word-break: break-all; color: var(--el-text-color-regular); line-height: 1.5">{{ row.catchup_url }}</span>
                </template>
              </div>
            </el-popover>
          </template>
        </el-table-column>
      </el-table>
      <div style="margin-top: 16px; display: flex; justify-content: flex-end">
        <el-pagination
          v-model:current-page="channelsPage"
          v-model:page-size="channelsPageSize"
          :page-sizes="[50, 100, 200, 500]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="filteredChannels.length"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { List, Refresh, Edit, Delete, Plus, SuccessFilled, CircleCloseFilled, Loading, Search } from '@element-plus/icons-vue'
import api from '../api'
import { usePolling } from '../composables/usePolling'
import ScheduleConfig from '../components/ScheduleConfig.vue'

const { t } = useI18n()

const sources = ref([])
const loading = ref(false)
const searchQuery = ref('')

const filteredSources = computed(() => {
  if (!searchQuery.value) return sources.value
  const q = searchQuery.value.toLowerCase()
  return sources.value.filter(s => s.name && s.name.toLowerCase().includes(q))
})
const { startPolling: startSyncPolling, stopPolling: stopSyncPolling } = usePolling(
  () => loadSources(false),
  3000
)

onUnmounted(() => {
  stopDetectPolling()
  stopCrackPolling()
})

const dialogVisible = ref(false)
const channelsVisible = ref(false)
const channels = ref([])
const channelsPage = ref(1)
const channelsPageSize = ref(100)
const channelsSourceId = ref(null)
const channelsSearch = ref('')

const filteredChannels = computed(() => {
  let result = channels.value
  if (channelsSearch.value) {
    const q = channelsSearch.value.toLowerCase()
    result = result.filter(ch => 
      (ch.tvg_id && ch.tvg_id.toLowerCase().includes(q)) || 
      (ch.name && ch.name.toLowerCase().includes(q))
    )
  }
  return result
})

const paginatedChannels = computed(() => {
  const start = (channelsPage.value - 1) * channelsPageSize.value
  const end = start + channelsPageSize.value
  return filteredChannels.value.slice(start, end)
})

function handleSearchChange() {
  channelsPage.value = 1
}

const detectedCount = computed(() => {
  return channels.value.filter(ch => ch.latency !== null && ch.latency !== undefined).length
})
const channelsDetecting = ref(false)
const detectTriggering = ref(false)
const detectStrategy = ref('unicast')
let detectPollingTimer = null
const isEdit = ref(false)
const editId = ref(null)
const submitting = ref(false)
const formRef = ref()
const epgStrategies = ref([])

// Crack dialog state
const crackDialogVisible = ref(false)
const crackAuthenticator = ref('')
const crackMode = ref('decimal')
const crackResult = ref(null)  // { key, fields }
const crackError = ref('')
const crackStopped = ref(false)
const crackProgress = ref(null) // { tried, total, percent }
const cracking = ref(false)
let crackPollTimer = null

const typeNameMap = computed(() => ({ iptv: 'IPTV', network_url: t('live_sources.type_network_url'), network_manual: t('live_sources.type_network_manual') }))
const typeTagMap = { iptv: 'danger', network_url: '', network_manual: 'warning' }

// Helper: split pipe-separated URLs into an array
function getUrls(row) {
  if (!row.url) return []
  return row.url.split('|').filter(u => u.trim())
}

// Helper: format schedule config JSON string into human-readable text
function formatSchedule(jsonStr) {
  if (!jsonStr) return '-'
  try {
    const cfg = JSON.parse(jsonStr)
    if (cfg.mode === 'interval' && cfg.hours) {
      return t('live_sources.schedule_mode_interval') + ' ' + cfg.hours + t('live_sources.schedule_hours_unit')
    }
    if (cfg.mode === 'daily' && cfg.times && cfg.times.length > 0) {
      return t('live_sources.schedule_mode_daily') + ' ' + cfg.times.join(', ')
    }
    return '-'
  } catch {
    return jsonStr || '-'
  }
}

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
  name: '', description: '', type: 'network_url', url: '', content: '', cron_time: '', cron_detect: '', detect_strategy: 'unicast',
  network_headers: [],
  epg_enabled: false, status: true,
  iptv: {
    platform: 'huawei',
    serverHost: '',
    providerSuffix: 'CTC',
    key: '',
    ip: '',
    epgStrategy: 'auto',
    headersList: defaultHeaders.map(h => ({ ...h })),
    authParamsStr: '',
  },
})
const form = reactive(defaultForm())

const formRules = computed(() => ({
  name: [{ required: true, message: t('common.required_name'), trigger: 'blur' }],
  type: [{ required: true, message: t('common.required_type'), trigger: 'change' }],
  url: [{ required: true, message: t('live_sources.required_url'), trigger: 'blur' }],
  content: [{ required: true, message: t('live_sources.required_content'), trigger: 'blur' }],
  'iptv.platform': [{ required: true, message: t('live_sources.required_platform'), trigger: 'change' }],
  'iptv.serverHost': [{ required: true, message: t('live_sources.required_server'), trigger: 'blur' }],
  'iptv.providerSuffix': [{ required: true, message: t('live_sources.required_provider'), trigger: 'change' }],
  'iptv.key': [{ required: true, message: t('live_sources.required_key'), trigger: 'blur' }],
  'iptv.ip': [{ required: true, message: t('live_sources.required_ip'), trigger: 'blur' }],
  'iptv.authParamsStr': [{ required: true, message: t('live_sources.required_auth_params'), trigger: 'blur' }],
}))

onMounted(async () => {
  await loadSources()
  try {
    const [epgRes] = await Promise.all([
      api.get('/settings/epg-strategies'),
    ])
    epgStrategies.value = epgRes.data
  } catch {}
})

async function loadSources(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    const { data } = await api.get('/live-sources')
    sources.value = data || []
    
    // Check polling
    const hasSyncing = sources.value.some(s => s.is_syncing)
    if (hasSyncing) {
      startSyncPolling()
    } else {
      stopSyncPolling()
    }
  } finally { if (showLoading) loading.value = false }
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
      throw new Error(t('live_sources.invalid_auth_json'))
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
    epgStrategy: iptv.epgStrategy || 'auto',
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
    epgStrategy: cfg.epgStrategy || cfg.channelProgramAPI || 'auto',
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
  
  let network_headers = []
  if (row.type === 'network_url' && row.headers) {
    try {
      const parsedHeaders = JSON.parse(row.headers)
      for (const [k, v] of Object.entries(parsedHeaders)) {
        network_headers.push({ name: k, value: v })
      }
    } catch {}
  }
  
  const iptvParsed = row.type === 'iptv' ? parseIptvConfig(row.iptv_config) : defaultForm().iptv
  Object.assign(form, {
    name: row.name, description: row.description || '', type: row.type, url: row.url, content: row.content,
    network_headers,
    cron_time: row.cron_time, cron_detect: row.cron_detect || '', detect_strategy: row.detect_strategy || 'unicast', status: row.status,
    iptv: iptvParsed,
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value.validate()
  submitting.value = true
  
  // Build headers for network_url (always send object so clearing headers works)
  let headersJson = null
  if (form.type === 'network_url') {
    const hdrs = {}
    if (form.network_headers) {
      for (const h of form.network_headers) {
        if (h.name && h.name.trim()) {
          hdrs[h.name.trim()] = h.value || ''
        }
      }
    }
    headersJson = hdrs
  }
  
  try {
    if (isEdit.value) {
      const body = { name: form.name, description: form.description, url: form.url, content: form.content, headers: headersJson, cron_time: form.cron_time || '', cron_detect: form.cron_detect || '', detect_strategy: form.detect_strategy, status: form.status }
      if (form.type === 'iptv') {
        body.iptv_config = buildIptvConfig()
      }
      if ((form.type === 'network_url' || form.type === 'network_manual') && form.epg_enabled) {
        body.auto_create_epg = true
      }
      const { data } = await api.put(`/live-sources/${editId.value}`, body)
      ElMessage.success(t('common.update_success'))
      if (data && data.warning) {
        ElMessage.warning(data.warning)
      }
    } else {
      const body = { name: form.name, description: form.description, type: form.type, url: form.url, content: form.content, headers: headersJson, cron_time: form.cron_time || '', cron_detect: form.cron_detect || '', detect_strategy: form.detect_strategy, epg_enabled: form.epg_enabled }
      if (form.type === 'iptv') {
        body.iptv_config = buildIptvConfig()
      }
      const { data } = await api.post('/live-sources', body)
      ElMessage.success(t('common.create_success'))
      if (data && data.warning) {
        ElMessage.warning(data.warning)
      }
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
    t('live_sources.delete_confirm', { name: row.name }),
    t('common.confirm_delete'),
    { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') }
  )
  await api.delete(`/live-sources/${row.id}`)
  ElMessage.success(t('common.delete_success'))
  await loadSources()
}

async function triggerFetch(row) {
  await api.post(`/live-sources/${row.id}/trigger`)
  ElMessage.success(t('common.trigger_success'))
  await loadSources(false)
}

async function showChannels(row) {
  try {
    channelsSourceId.value = row.id
    const sourceRes = await api.get(`/live-sources/${row.id}`)
    channelsDetecting.value = sourceRes.data.is_detecting || false
    const { data } = await api.get(`/live-sources/${row.id}/channels`)
    channels.value = data.channels || []
    channelsPage.value = 1
    channelsSearch.value = ''
    channelsVisible.value = true

    // Start polling if detecting
    if (channelsDetecting.value) {
      startDetectPolling()
    }
  } catch {}
}

async function triggerDetect() {
  if (!channelsSourceId.value) return
  detectTriggering.value = true
  try {
    await api.post(`/live-sources/${channelsSourceId.value}/detect`, { detect_strategy: detectStrategy.value })
    ElMessage.success(t('live_sources.trigger_detect'))
    channelsDetecting.value = true
    
    // Reset local channels state so UI immediately shows loading state
    channels.value.forEach(ch => {
      ch.latency = null
      ch.video_codec = null
      ch.video_resolution = null
    })
    
    startDetectPolling()
    await loadSources(false)
  } catch (e) {
    if (e.response?.data?.error) {
      ElMessage.error(e.response.data.error)
    }
  } finally {
    detectTriggering.value = false
  }
}

function startDetectPolling() {
  if (detectPollingTimer) return
  detectPollingTimer = setInterval(async () => {
    if (!channelsSourceId.value || !channelsVisible.value) {
      stopDetectPolling()
      return
    }
    try {
      // Check source detecting status
      const sourceRes = await api.get(`/live-sources/${channelsSourceId.value}`)
      const isDetecting = sourceRes.data.is_detecting || false

      // Refresh channel data
      const { data } = await api.get(`/live-sources/${channelsSourceId.value}/channels`)
      channels.value = data.channels || []

      if (!isDetecting) {
        channelsDetecting.value = false
        stopDetectPolling()
        await loadSources(false)
      }
    } catch {
      stopDetectPolling()
    }
  }, 3000)
}

function stopDetectPolling() {
  if (detectPollingTimer) {
    clearInterval(detectPollingTimer)
    detectPollingTimer = null
  }
}

function fillAuthExample() {
  form.iptv.authParamsStr = authParamsExample
}

// Crack key - polling based
function stopCrackPolling() {
  if (crackPollTimer) {
    clearInterval(crackPollTimer)
    crackPollTimer = null
  }
}

function startCrackPolling() {
  stopCrackPolling()
  crackPollTimer = setInterval(async () => {
    try {
      const { data } = await api.get('/crack-key/status')
      applyCrackStatus(data)
    } catch {
      stopCrackPolling()
    }
  }, 1000)
}

function applyCrackStatus(status) {
  if (status.state === 'running') {
    cracking.value = true
    crackStopped.value = false
    crackError.value = ''
    crackResult.value = null
    crackAuthenticator.value = status.authenticator || ''
    crackMode.value = status.mode || 'decimal'
    if (status.progress) {
      crackProgress.value = status.progress
    }
  } else if (status.state === 'completed') {
    cracking.value = false
    crackResult.value = status.result
    crackProgress.value = status.progress
    crackAuthenticator.value = status.authenticator || ''
    crackMode.value = status.mode || 'decimal'
    stopCrackPolling()
  } else if (status.state === 'failed') {
    cracking.value = false
    crackError.value = status.error || t('live_sources.crack_failed')
    crackProgress.value = status.progress
    stopCrackPolling()
  } else if (status.state === 'stopped') {
    cracking.value = false
    crackStopped.value = true
    crackProgress.value = status.progress
    stopCrackPolling()
  } else {
    // idle
    cracking.value = false
    stopCrackPolling()
  }
}

async function showCrackDialog() {
  // Reset local UI state
  crackAuthenticator.value = ''
  crackMode.value = 'decimal'
  crackResult.value = null
  crackError.value = ''
  crackStopped.value = false
  crackProgress.value = null
  cracking.value = false

  // Check if a task is already running or completed
  try {
    const { data } = await api.get('/crack-key/status')
    if (data.state !== 'idle') {
      applyCrackStatus(data)
      if (data.state === 'running') {
        startCrackPolling()
      }
    }
  } catch {}

  crackDialogVisible.value = true
}

async function doCrack() {
  if (!crackAuthenticator.value.trim()) {
    ElMessage.warning(t('live_sources.crack_enter_auth'))
    return
  }
  cracking.value = true
  crackResult.value = null
  crackError.value = ''
  crackStopped.value = false
  crackProgress.value = null

  try {
    await api.post('/crack-key/start', {
      authenticator: crackAuthenticator.value.trim(),
      mode: crackMode.value
    })
    startCrackPolling()
  } catch (e) {
    cracking.value = false
    crackError.value = e.response?.data?.error || t('live_sources.crack_failed')
  }
}

async function stopCrack() {
  try {
    await api.post('/crack-key/stop')
  } catch {}
  crackStopped.value = true
  cracking.value = false
  stopCrackPolling()
}

function onCrackDialogClose() {
  // Only stop polling, do NOT stop backend task
  stopCrackPolling()
}

function applyCrackResult() {
  if (crackResult.value) {
    form.iptv.key = crackResult.value.key
    crackDialogVisible.value = false
    ElMessage.success(t('live_sources.key_applied'))
  }
}

function formatNumber(num) {
  if (num === null || num === undefined) return '0'
  return num.toLocaleString()
}
</script>