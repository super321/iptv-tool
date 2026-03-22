<template>
  <div class="settings-page">
    <h3 style="margin: 0 0 20px">{{ $t('settings_about.title') }}</h3>

    <el-card shadow="hover" class="settings-card">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <el-icon :size="18"><InfoFilled /></el-icon>
          <span>{{ $t('settings_about.title') }}</span>
        </div>
      </template>
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item :label="$t('settings_about.system_name')">IPTV Tool</el-descriptions-item>
        <el-descriptions-item :label="$t('settings_about.system_version')">
          <div class="version-cell">
            <div class="version-cell-top">
              <el-tag v-if="appVersion" size="small" type="primary">{{ appVersion }}</el-tag>
              <span v-else style="color: #909399; font-size: 12px">{{ $t('settings_about.fetching') }}</span>

              <!-- Update status hints -->
              <span v-if="updateStatus === 'checking'" class="update-hint">
                <el-icon class="is-loading" :size="12"><Loading /></el-icon>
                {{ $t('settings_about.checking_update') }}
              </span>
              <el-tag v-else-if="updateStatus === 'latest'" size="small" type="success" effect="plain" class="update-status-tag">
                <span style="display: flex; align-items: center; gap: 2px">
                  <el-icon :size="12"><CircleCheckFilled /></el-icon>
                  <span>{{ $t('settings_about.already_latest') }}</span>
                </span>
              </el-tag>
              <el-link v-else-if="updateStatus === 'available'" type="warning" :underline="false" class="update-available-link" @click="showUpdateDialog = true">
                <span style="display: flex; align-items: center; gap: 2px">
                  <el-icon :size="14"><TopRight /></el-icon>
                  <span>{{ $t('settings_about.new_version_available') }}: {{ latestVersion }}</span>
                </span>
              </el-link>

              <el-button
                size="small"
                :icon="Refresh"
                :loading="manualChecking"
                @click="manualCheckUpdate"
                circle
                :title="$t('settings_about.check_update')"
              />
            </div>
          </div>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('settings_about.tech_stack')">Vue 3 + Element Plus / Go + Gin + SQLite</el-descriptions-item>
        <el-descriptions-item :label="$t('settings_about.runtime_mode')">{{ $t('settings_about.runtime_value') }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="hover" class="settings-card star-card">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <el-icon :size="18"><Star /></el-icon>
          <span>{{ $t('settings_about.star_title') }}</span>
        </div>
      </template>

      <div class="star-content">
        <div class="star-info">
          <img src="/iptv-tool.svg" alt="IPTV Tool" class="star-logo" />
          <p class="star-desc">{{ $t('settings_about.star_desc') }}</p>
        </div>
        <el-button type="warning" class="star-btn" @click="openLink('https://github.com/super321/iptv-tool')">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" style="margin-right: 6px">
            <path d="M12 .587l3.668 7.568L24 9.306l-6 5.924 1.416 8.356L12 19.446l-7.416 4.14L6 15.23 0 9.306l8.332-1.151z"/>
          </svg>
          {{ $t('settings_about.star_btn') }}
        </el-button>
      </div>
    </el-card>

    <el-card shadow="hover" class="settings-card sponsor-card">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <el-icon :size="18"><Coffee /></el-icon>
          <span>{{ $t('settings_about.sponsor') }}</span>
        </div>
      </template>

      <p class="sponsor-desc">{{ $t('settings_about.sponsor_desc') }}</p>

      <div class="sponsor-list">
        <!-- Ko-fi -->
        <div class="sponsor-item">
          <div class="sponsor-info">
            <div class="sponsor-icon kofi-icon">
              <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
                <path d="M23.881 8.948c-.773-4.085-4.859-4.593-4.859-4.593H.723c-.604 0-.679.798-.679.798s-.082 7.324-.022 11.822c.164 2.424 2.586 2.672 2.586 2.672s8.267-.023 11.966-.049c2.438-.426 2.683-2.566 2.658-3.734 4.352.24 7.422-2.831 6.649-6.916zm-11.062 3.511c-1.246 1.453-4.011 3.976-4.011 3.976s-.121.119-.31.023c-.076-.057-.108-.09-.108-.09-.443-.441-3.368-3.049-4.034-3.954-.709-.965-1.041-2.7-.091-3.71.951-1.01 3.005-1.086 4.363.407 0 0 1.565-1.782 3.468-.963 1.904.82 1.832 3.011.723 4.311zm6.173.478c-.928.116-1.682.028-1.682.028V7.284h1.77s1.971.551 1.971 2.638c0 1.913-.985 2.667-2.059 3.015z"/>
              </svg>
            </div>
            <div class="sponsor-text">
              <span class="sponsor-name">Ko-fi</span>
              <span class="sponsor-sub">Buy me a coffee</span>
            </div>
          </div>
          <el-button type="warning" class="kofi-btn" @click="openLink('https://ko-fi.com/super321')">
            <el-icon style="margin-right: 4px"><Link /></el-icon>
            {{ $t('settings_about.go_sponsor') }}
          </el-button>
        </div>

        <!-- 微信赞赏码 -->
        <div class="sponsor-item">
          <div class="sponsor-info">
            <div class="sponsor-icon wechat-icon">
              <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
                <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 01.213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 00.167-.054l1.903-1.114a.864.864 0 01.717-.098 10.16 10.16 0 002.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178A1.17 1.17 0 014.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178 1.17 1.17 0 01-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.942 2.453 3.666 4.229 6.884 4.229.826 0 1.622-.12 2.361-.336a.722.722 0 01.598.082l1.584.926a.272.272 0 00.14.045c.134 0 .24-.111.24-.245 0-.06-.024-.12-.04-.178l-.325-1.233a.492.492 0 01.177-.554C23.028 18.48 24 16.82 24 14.98c0-3.21-2.874-5.952-7.062-6.122zm-2.07 2.999c.535 0 .969.44.969.982a.976.976 0 01-.969.983.976.976 0 01-.969-.983c0-.542.434-.982.97-.982zm4.827 0c.535 0 .969.44.969.982a.976.976 0 01-.969.983.976.976 0 01-.969-.983c0-.542.434-.982.97-.982z"/>
              </svg>
            </div>
            <div class="sponsor-text">
              <span class="sponsor-name">{{ $t('settings_about.wechat_reward') }}</span>
              <span class="sponsor-sub">{{ $t('settings_about.wechat_reward_sub') }}</span>
            </div>
          </div>
          <el-button type="success" class="wechat-btn" @click="showWechatDialog = true">
            <el-icon style="margin-right: 4px"><View /></el-icon>
            {{ $t('settings_about.view_qrcode') }}
          </el-button>
        </div>

        <!-- Ethereum -->
        <div class="sponsor-item eth-item">
          <div class="sponsor-info">
            <div class="sponsor-icon eth-icon">
              <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
                <path d="M12 1.5l-7 10.657L12 16.5l7-4.343L12 1.5zM12 17.657l-7-4.343L12 22.5l7-9.186-7 4.343z"/>
              </svg>
            </div>
            <div class="sponsor-text">
              <span class="sponsor-name">Ethereum</span>
              <span class="sponsor-sub">{{ $t('settings_about.eth_sub') }}</span>
            </div>
          </div>
          <el-button type="info" @click="showEthDialog = true">
            <el-icon style="margin-right: 4px"><Wallet /></el-icon>
            {{ $t('settings_about.view_address') }}
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- WeChat Reward Dialog -->
    <el-dialog v-model="showWechatDialog" :title="$t('settings_about.wechat_reward_title')" width="560px" align-center>
      <div class="wechat-dialog-content">
        <img :src="wechatQrCode" alt="WeChat Reward QR Code" class="wechat-qrcode" />
      </div>
    </el-dialog>

    <!-- Ethereum Detail Dialog -->
    <el-dialog v-model="showEthDialog" :title="$t('settings_about.eth_title')" width="560px" align-center>
      <div class="eth-dialog-content">
        <img :src="ethQrCode" alt="Ethereum QR Code" class="eth-qrcode" />
        <p class="eth-note">{{ $t('settings_about.eth_note') }}</p>
        <div class="eth-address-box">
          <span class="eth-label">{{ $t('settings_about.wallet_address') }}</span>
          <el-input
            v-model="ethAddress"
            readonly
            size="small"
          >
            <template #append>
              <el-button @click="copyAddress">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </template>
          </el-input>
        </div>
      </div>
    </el-dialog>

    <!-- Update Dialog -->
    <el-dialog v-model="showUpdateDialog" :title="$t('settings_about.new_version_title')" width="520px" align-center>
      <div class="update-dialog-content">
        <div class="update-version-info">
          <div class="version-row">
            <span class="version-label">{{ $t('settings_about.current_version_label') }}</span>
            <el-tag size="small" type="info">{{ appVersion }}</el-tag>
          </div>
          <div class="version-row">
            <span class="version-label">{{ $t('settings_about.latest_version') }}</span>
            <el-tag size="small" type="success">{{ latestVersion }}</el-tag>
          </div>
        </div>
        <div v-if="releaseNotes" class="release-notes-section">
          <div class="release-notes-label">{{ $t('settings_about.release_notes') }}</div>
          <div class="release-notes-content markdown-body" v-html="renderedNotes"></div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showUpdateDialog = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="openLink(releaseUrl)">
          <el-icon style="margin-right: 4px"><Link /></el-icon>
          {{ $t('settings_about.go_download') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { InfoFilled, Link, CopyDocument, Wallet, Refresh, Loading, CircleCheckFilled, TopRight, View, Star } from '@element-plus/icons-vue'
import { Coffee } from '@element-plus/icons-vue'
import { marked } from 'marked'
import ethQrCode from '../assets/eth_qrcode.jpg'
import wechatQrCode from '../assets/wechat_qrcode.jpg'
import api from '../api'

const { t } = useI18n()

const showWechatDialog = ref(false)
const showEthDialog = ref(false)
const ethAddress = ref('0x6989acE6Eb2CC196fAFce3cEcAEC6b6b63716C83')
const appVersion = ref('')

// Update check state
const updateStatus = ref('') // '', 'checking', 'latest', 'available'
const latestVersion = ref('')
const releaseNotes = ref('')
const releaseUrl = ref('')
const showUpdateDialog = ref(false)
const manualChecking = ref(false)

// Rendered markdown for release notes
const renderedNotes = computed(() => {
  if (!releaseNotes.value) return ''
  return marked(releaseNotes.value, { breaks: true })
})

// Cache keys for localStorage
const CACHE_KEY_VERSION = 'iptv_update_latest_version'
const CACHE_KEY_NOTES = 'iptv_update_release_notes'
const CACHE_KEY_URL = 'iptv_update_release_url'
const CACHE_KEY_TIME = 'iptv_update_check_time'
const CACHE_TTL = 60 * 60 * 1000 // 1 hour in milliseconds

onMounted(async () => {
  // Fetch current version
  try {
    const { data } = await api.get('/system/version')
    appVersion.value = data.version || 'unknown'
  } catch {
    appVersion.value = 'unknown'
  }

  // Auto-check for updates (respecting cache TTL)
  autoCheckUpdate()
})

function isCacheValid() {
  const lastCheck = localStorage.getItem(CACHE_KEY_TIME)
  if (!lastCheck) return false
  return (Date.now() - parseInt(lastCheck, 10)) < CACHE_TTL
}

function loadFromCache() {
  latestVersion.value = localStorage.getItem(CACHE_KEY_VERSION) || ''
  releaseNotes.value = localStorage.getItem(CACHE_KEY_NOTES) || ''
  releaseUrl.value = localStorage.getItem(CACHE_KEY_URL) || ''
}

function saveToCache(version, notes, url) {
  localStorage.setItem(CACHE_KEY_VERSION, version)
  localStorage.setItem(CACHE_KEY_NOTES, notes)
  localStorage.setItem(CACHE_KEY_URL, url)
  localStorage.setItem(CACHE_KEY_TIME, Date.now().toString())
}

function compareVersions(current, latest) {
  const stripV = v => v.replace(/^v/, '').trim()
  const c = stripV(current)
  const l = stripV(latest)
  if (c === 'dev') return l === 'dev' ? 0 : -1
  if (l === 'dev') return 1
  const cp = c.split('.').map(Number)
  const lp = l.split('.').map(Number)
  const maxLen = Math.max(cp.length, lp.length)
  for (let i = 0; i < maxLen; i++) {
    const cv = cp[i] || 0
    const lv = lp[i] || 0
    if (cv < lv) return -1
    if (cv > lv) return 1
  }
  return 0
}

function applyUpdateResult(current, latest, notes, url) {
  latestVersion.value = latest
  releaseNotes.value = notes
  releaseUrl.value = url

  if (compareVersions(current, latest) < 0) {
    updateStatus.value = 'available'
  } else {
    updateStatus.value = 'latest'
  }
}

async function autoCheckUpdate() {
  // If cache is still valid, use cached data
  if (isCacheValid()) {
    loadFromCache()
    const current = appVersion.value || 'dev'
    if (latestVersion.value) {
      if (compareVersions(current, latestVersion.value) < 0) {
        updateStatus.value = 'available'
      } else {
        updateStatus.value = 'latest'
      }
    }
    return
  }

  // Fetch from API
  await fetchUpdateInfo(false)
}

async function manualCheckUpdate() {
  manualChecking.value = true
  await fetchUpdateInfo(true)
  manualChecking.value = false
}

async function fetchUpdateInfo(isManual) {
  updateStatus.value = 'checking'

  try {
    const { data } = await api.get('/system/check-update')
    saveToCache(data.latest_version, data.release_notes || '', data.release_url || '')
    applyUpdateResult(
      data.current_version,
      data.latest_version,
      data.release_notes || '',
      data.release_url || ''
    )

    if (isManual) {
      if (data.has_update) {
        showUpdateDialog.value = true
      } else {
        ElMessage.success(t('settings_about.already_latest'))
      }
    }
  } catch {
    updateStatus.value = ''
    if (isManual) {
      ElMessage.error(t('settings_about.check_failed'))
    }
  }
}

function openLink(url) {
  window.open(url, '_blank')
}

async function copyAddress() {
  try {
    await navigator.clipboard.writeText(ethAddress.value)
    ElMessage.success(t('settings_about.copied'))
  } catch {
    ElMessage.error(t('settings_about.copy_failed'))
  }
}
</script>

<style scoped>
.settings-page {
  width: 100%;
}
.settings-card {
  max-width: 600px;
}
.star-card {
  margin-top: 20px;
}
.star-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
}
.star-info {
  display: flex;
  align-items: center;
  gap: 14px;
  flex: 1;
}
.star-logo {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
}
.star-desc {
  margin: 0;
  color: #606266;
  font-size: 14px;
  line-height: 1.6;
  flex: 1;
}
.star-btn {
  background: linear-gradient(135deg, #f5af19, #f09819) !important;
  border: none !important;
  color: #fff !important;
  font-weight: 600;
  flex-shrink: 0;
  margin-left: 16px;
}
.star-btn:hover {
  opacity: 0.9;
}
.sponsor-card {
  margin-top: 20px;
}
.sponsor-desc {
  margin: 0 0 20px;
  color: #606266;
  font-size: 14px;
  line-height: 1.6;
}

/* Version cell layout */
.version-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.version-cell-top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* Update hints */
.update-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #909399;
}
.update-available-link {
  font-size: 12px !important;
  font-weight: 500;
}

/* Update dialog */
.update-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.update-version-info {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 16px;
  background: #f5f7fa;
  border-radius: 8px;
}
.version-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.version-label {
  font-size: 13px;
  color: #606266;
  min-width: 70px;
}
.release-notes-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.release-notes-label {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}
.release-notes-content {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 12px 16px;
  font-size: 13px;
  line-height: 1.7;
  color: #606266;
  word-break: break-word;
  max-height: 300px;
  overflow-y: auto;
}

/* Markdown body styles */
.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  margin: 12px 0 6px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}
.markdown-body :deep(h1) { font-size: 16px; }
.markdown-body :deep(h2) { font-size: 15px; }
.markdown-body :deep(p) {
  margin: 4px 0;
}
.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  margin: 4px 0;
  padding-left: 20px;
}
.markdown-body :deep(li) {
  margin: 2px 0;
}
.markdown-body :deep(code) {
  background: #e8eaed;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 12px;
}
.markdown-body :deep(pre) {
  background: #e8eaed;
  padding: 8px 12px;
  border-radius: 6px;
  overflow-x: auto;
}
.markdown-body :deep(pre code) {
  background: none;
  padding: 0;
}
.markdown-body :deep(a) {
  color: #409eff;
  text-decoration: none;
}
.markdown-body :deep(a:hover) {
  text-decoration: underline;
}
.markdown-body :deep(blockquote) {
  margin: 4px 0;
  padding: 4px 12px;
  border-left: 3px solid #dcdfe6;
  color: #909399;
}
.markdown-body :deep(> :first-child) {
  margin-top: 0;
}
.markdown-body :deep(> :last-child) {
  margin-bottom: 0;
}

/* Sponsor list */
.sponsor-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.sponsor-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-radius: 8px;
  background: #f9fafb;
  transition: background 0.2s;
}
.sponsor-item:hover {
  background: #f0f2f5;
}
.sponsor-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.sponsor-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex-shrink: 0;
}
.kofi-icon {
  background: linear-gradient(135deg, #ff5e5b, #ff4081);
}
.wechat-icon {
  background: linear-gradient(135deg, #2aae67, #07c160);
}
.eth-icon {
  background: linear-gradient(135deg, #627eea, #3c3c3d);
}
.sponsor-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.sponsor-name {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}
.sponsor-sub {
  font-size: 12px;
  color: #909399;
}

/* Buttons */
.kofi-btn {
  background: linear-gradient(135deg, #ff5e5b, #ff4081) !important;
  border: none !important;
  color: #fff !important;
}
.kofi-btn:hover {
  opacity: 0.9;
}
.wechat-btn {
  background: linear-gradient(135deg, #2aae67, #07c160) !important;
  border: none !important;
  color: #fff !important;
}
.wechat-btn:hover {
  opacity: 0.9;
}

/* WeChat Reward Dialog */
.wechat-dialog-content {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.wechat-qrcode {
  width: 480px;
  height: 480px;
  object-fit: contain;
  border-radius: 8px;
}

/* Ethereum Dialog */
.eth-dialog-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.eth-qrcode {
  width: 480px;
  height: 480px;
  object-fit: contain;
  border-radius: 8px;
}
.eth-note {
  margin: 0;
  font-size: 13px;
  color: #909399;
}
.eth-address-box {
  width: 100%;
  text-align: center;
}
.eth-label {
  display: block;
  margin-bottom: 8px;
  font-size: 13px;
  color: #606266;
}
</style>
