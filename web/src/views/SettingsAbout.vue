<template>
  <div class="settings-page">
    <h3 style="margin: 0 0 20px">关于系统</h3>

    <el-card shadow="hover" class="settings-card">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <el-icon :size="18"><InfoFilled /></el-icon>
          <span>关于系统</span>
        </div>
      </template>
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item label="系统名称">IPTV Tool</el-descriptions-item>
        <el-descriptions-item label="技术栈">Vue 3 + Element Plus / Go + Gin + SQLite</el-descriptions-item>
        <el-descriptions-item label="运行模式">单文件部署 (go:embed)</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="hover" class="settings-card sponsor-card">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <el-icon :size="18"><Coffee /></el-icon>
          <span>赞助支持</span>
        </div>
      </template>

      <p class="sponsor-desc">如果你觉得这个项目对你有帮助，可以考虑赞助支持开发者，感谢你的慷慨！</p>

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
            前往赞助
          </el-button>
        </div>

        <!-- 爱发电 -->
        <div class="sponsor-item">
          <div class="sponsor-info">
            <div class="sponsor-icon afdian-icon">
              <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
                <path d="M12 2L4 7v10l8 5 8-5V7l-8-5zm0 2.18L18 8v8l-6 3.82L6 16V8l6-3.82z"/>
                <path d="M12 8l-4 2.5v5L12 18l4-2.5v-5L12 8zm0 2.18l2 1.25v2.5L12 15.18l-2-1.25v-2.5L12 10.18z"/>
              </svg>
            </div>
            <div class="sponsor-text">
              <span class="sponsor-name">爱发电</span>
              <span class="sponsor-sub">国内赞助平台</span>
            </div>
          </div>
          <el-button type="primary" class="afdian-btn" @click="openLink('https://afdian.com/a/super321')">
            <el-icon style="margin-right: 4px"><Link /></el-icon>
            前往赞助
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
              <span class="sponsor-sub">ETH / ERC20 代币</span>
            </div>
          </div>
          <el-button type="info" @click="showEthDialog = true">
            <el-icon style="margin-right: 4px"><Wallet /></el-icon>
            查看地址
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- Ethereum Detail Dialog -->
    <el-dialog v-model="showEthDialog" title="Ethereum 赞助" width="400px" align-center>
      <div class="eth-dialog-content">
        <img :src="ethQrCode" alt="Ethereum QR Code" class="eth-qrcode" />
        <p class="eth-note">仅支持 Ethereum 资产 (ERC20)</p>
        <div class="eth-address-box">
          <span class="eth-label">钱包地址</span>
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
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { InfoFilled, Link, CopyDocument, Wallet } from '@element-plus/icons-vue'
import { Coffee } from '@element-plus/icons-vue'
import ethQrCode from '../assets/eth_qrcode.jpg'

const showEthDialog = ref(false)
const ethAddress = ref('0x6989acE6Eb2CC196fAFce3cEcAEC6b6b63716C83')

function openLink(url) {
  window.open(url, '_blank')
}

async function copyAddress() {
  try {
    await navigator.clipboard.writeText(ethAddress.value)
    ElMessage.success('地址已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败，请手动复制')
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
.sponsor-card {
  margin-top: 20px;
}
.sponsor-desc {
  margin: 0 0 20px;
  color: #606266;
  font-size: 14px;
  line-height: 1.6;
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
.afdian-icon {
  background: linear-gradient(135deg, #946ce6, #7c4dff);
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
.afdian-btn {
  background: linear-gradient(135deg, #946ce6, #7c4dff) !important;
  border: none !important;
  color: #fff !important;
}
.afdian-btn:hover {
  opacity: 0.9;
}

/* Ethereum Dialog */
.eth-dialog-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.eth-qrcode {
  width: 220px;
  height: 220px;
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
