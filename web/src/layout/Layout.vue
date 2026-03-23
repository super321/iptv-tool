<template>
  <el-container style="height: 100vh">
    <el-aside :width="isCollapsed ? '64px' : '220px'" class="aside-menu">
      <div class="logo-area">
        <div class="logo-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <transition name="fade">
          <span v-if="!isCollapsed" class="logo-text">IPTV Tool</span>
        </transition>
      </div>
      <el-menu
          :default-active="activeMenu"
          :collapse="isCollapsed"
          class="custom-menu"
          :collapse-transition="false"
          router
      >
        <el-menu-item index="/live-sources">
          <el-icon><VideoCamera /></el-icon>
          <template #title>{{ $t('nav.live_sources') }}</template>
        </el-menu-item>
        <el-menu-item index="/epg-sources">
          <el-icon><Calendar /></el-icon>
          <template #title>{{ $t('nav.epg_sources') }}</template>
        </el-menu-item>
        <el-menu-item index="/logos">
          <el-icon><Picture /></el-icon>
          <template #title>{{ $t('nav.logos') }}</template>
        </el-menu-item>
        <el-menu-item index="/rules">
          <el-icon><Guide /></el-icon>
          <template #title>{{ $t('nav.rules') }}</template>
        </el-menu-item>
        <el-menu-item index="/publish">
          <el-icon><Share /></el-icon>
          <template #title>{{ $t('nav.publish') }}</template>
        </el-menu-item>
        <el-sub-menu index="/logs">
          <template #title>
            <el-icon><Document /></el-icon>
            <span>{{ $t('nav.logs') }}</span>
          </template>
          <el-menu-item index="/logs/runtime">
            <el-icon><Monitor /></el-icon>
            <template #title>{{ $t('nav.logs_runtime') }}</template>
          </el-menu-item>
          <el-menu-item index="/logs/access">
            <el-icon><Connection /></el-icon>
            <template #title>{{ $t('nav.logs_access') }}</template>
          </el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="/settings">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>{{ $t('nav.settings') }}</span>
          </template>
          <el-menu-item index="/settings/detect">
            <el-icon><Stopwatch /></el-icon>
            <template #title>{{ $t('nav.detect') }}</template>
          </el-menu-item>
          <el-menu-item index="/settings/access-control">
            <el-icon><Lock /></el-icon>
            <template #title>{{ $t('settings_access_control.nav_label') }}</template>
          </el-menu-item>
          <el-menu-item index="/settings/password">
            <el-icon><Key /></el-icon>
            <template #title>{{ $t('nav.password') }}</template>
          </el-menu-item>
          <el-menu-item index="/settings/about">
            <el-icon><InfoFilled /></el-icon>
            <template #title>{{ $t('nav.about') }}</template>
          </el-menu-item>
        </el-sub-menu>
      </el-menu>
      <div class="collapse-btn" @click="isCollapsed = !isCollapsed">
        <el-icon :size="18">
          <Fold v-if="!isCollapsed" />
          <Expand v-else />
        </el-icon>
      </div>
    </el-aside>
    <el-container>
      <el-header class="top-header">
        <el-breadcrumb separator="/" style="margin-right: auto">
          <el-breadcrumb-item>{{ $t('nav.admin_system') }}</el-breadcrumb-item>
        </el-breadcrumb>
        
        <div class="header-right">
          <el-dropdown @command="switchLanguage" class="header-item">
            <span class="user-dropdown icon-only">
              <svg class="lang-icon" viewBox="0 0 24 24" width="20" height="20">
                <text x="2" y="15" font-size="14" font-weight="600" font-family="sans-serif" fill="currentColor">文</text>
                <text x="14" y="20" font-size="10" font-weight="700" font-family="sans-serif" fill="currentColor">A</text>
              </svg>
              <el-icon style="margin-left: 2px" :size="12"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="zh" :class="{ 'is-active': currentLocale === 'zh' }">{{ $t('language.zh') }}</el-dropdown-item>
                <el-dropdown-item command="zh-Hant" :class="{ 'is-active': currentLocale === 'zh-Hant' }">{{ $t('language.zh-Hant') }}</el-dropdown-item>
                <el-dropdown-item command="en" :class="{ 'is-active': currentLocale === 'en' }">{{ $t('language.en') }}</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <div class="header-item theme-switch-wrapper">
             <el-switch
              v-model="isDarkTheme"
              inline-prompt
              :active-icon="Moon"
              :inactive-icon="Sunny"
              class="custom-theme-switch"
              style="--el-switch-on-color: var(--el-fill-color-light); --el-switch-off-color: var(--el-fill-color-light); --el-switch-border-color: var(--el-border-color-lighter)"
            />
          </div>

          <el-dropdown @command="handleCommand" class="header-item">
            <span class="user-dropdown icon-only">
              <el-icon :size="20"><User /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item disabled>
                  <el-icon><UserFilled /></el-icon>{{ currentUsername }}
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>{{ $t('nav.logout') }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <el-main style="padding: 24px; background: var(--main-bg)">
        <router-view v-slot="{ Component }">
          <transition name="fade-transform" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>
<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useThemeStore } from '../stores/theme'
import { loadLocale } from '../i18n'
import { UserFilled, User, SwitchButton, VideoCamera, Monitor, Calendar, Picture, Guide, Share, Setting, Fold, Expand, ArrowDown, Lock, InfoFilled, Stopwatch, Key, Document, Connection, Sunny, Moon } from '@element-plus/icons-vue'
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const themeStore = useThemeStore()
const { t, locale } = useI18n()
const currentLocale = computed(() => locale.value)
const isCollapsed = ref(false)
const activeMenu = computed(() => route.path)

const isDarkTheme = computed({
  get: () => themeStore.isDark,
  set: (val) => themeStore.setMode(val ? 'dark' : 'light')
})

// Extract username from JWT token payload
const currentUsername = computed(() => {
  try {
    const token = auth.token
    if (!token) return ''
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.username || ''
  } catch {
    return ''
  }
})

function handleCommand(cmd) {
  if (cmd === 'logout') {
    auth.logout()
    router.push('/login')
  }
}
async function switchLanguage(lang) {
  await loadLocale(lang)
}
</script>
<style scoped>
.aside-menu {
  background: var(--sidebar-bg);
  transition: width 0.3s, background 0.3s;
  display: flex;
  flex-direction: column;
  box-shadow: 2px 0 8px var(--sidebar-shadow);
  z-index: 10;
}
.logo-area {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
  background: var(--sidebar-logo-bg);
  overflow: hidden;
  white-space: nowrap;
}
.logo-icon {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #409eff 0%, #337ecc 100%);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
  margin-right: 8px;
}
.logo-text {
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 0.5px;
}
.custom-menu {
  flex: 1;
  border-right: none;
  background-color: transparent;
  --el-menu-bg-color: transparent;
  --el-menu-hover-bg-color: rgba(255, 255, 255, 0.05);
  --el-menu-text-color: #bfcbd9;
  --el-menu-active-color: #409eff;
}
:deep(.el-menu-item.is-active) {
  background-color: rgba(64, 158, 255, 0.1) !important;
  border-right: 3px solid #409eff;
}
.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 50px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: var(--sidebar-collapse-bg);
  transition: all 0.3s;
  overflow: hidden;
  white-space: nowrap;
}
.collapse-btn:hover {
  color: #fff;
  background: var(--sidebar-collapse-hover-bg);
}
.top-header {
  background: var(--header-bg);
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 1px 4px var(--header-shadow);
  padding: 0 20px;
  height: 60px;
  backdrop-filter: blur(10px);
  transition: background 0.3s, box-shadow 0.3s;
}
.header-right {
  display: flex;
  align-items: center;
  height: 100%;
}
.header-item {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  outline: none;
}
/* Remove el-dropdown focus outline/border */
:deep(.header-item .el-tooltip__trigger:focus-visible) {
  outline: none;
}
.header-right :deep(.el-dropdown:focus-visible),
.header-right :deep(.el-dropdown__caret-button:focus-visible) {
  outline: none;
}
.user-dropdown {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: var(--el-text-color-regular);
  font-weight: 500;
  transition: color 0.3s;
  outline: none;
}
.user-dropdown.icon-only {
  font-size: 18px;
}
.user-dropdown:hover {
  color: #409eff;
}
.user-dropdown:focus {
  outline: none;
}
.lang-icon {
  color: var(--el-text-color-regular);
  transition: color 0.3s;
}
.user-dropdown:hover .lang-icon {
  color: #409eff;
}
.theme-switch-wrapper {
  padding: 0 16px;
}
.theme-switch-wrapper:hover :deep(.el-switch__core) {
  box-shadow: 0 0 8px rgba(64, 158, 255, 0.4);
}
/* Custom Element Plus Switch styling to match the screenshot */
.custom-theme-switch {
  --el-switch-on-color: var(--el-fill-color-light);
  --el-switch-off-color: var(--el-fill-color-light);
  --el-switch-border-color: var(--el-border-color-lighter);
}
:deep(.custom-theme-switch .el-switch__core) {
  border: 1px solid var(--el-border-color-lighter);
  background-color: var(--el-fill-color-light) !important;
  transition: box-shadow 0.3s;
}
:deep(.custom-theme-switch .el-switch__action) {
  background-color: var(--el-bg-color-overlay);
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  color: var(--el-text-color-regular);
  display: flex;
  align-items: center;
  justify-content: center;
}
:deep(.custom-theme-switch.is-checked .el-switch__action) {
  color: var(--el-text-color-primary);
}
:deep(.custom-theme-switch .el-switch__inner .el-icon) {
  color: var(--el-text-color-secondary);
}
/* Animations */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
.fade-transform-enter-active,
.fade-transform-leave-active {
  transition: all 0.3s ease;
}
.fade-transform-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}
.fade-transform-leave-to {
  opacity: 0;
  transform: translateX(20px);
}
</style>