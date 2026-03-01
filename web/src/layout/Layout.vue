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
          <template #title>直播源</template>
        </el-menu-item>
        <el-menu-item index="/epg-sources">
          <el-icon><Calendar /></el-icon>
          <template #title>EPG源</template>
        </el-menu-item>
        <el-menu-item index="/logos">
          <el-icon><Picture /></el-icon>
          <template #title>台标管理</template>
        </el-menu-item>
        <el-menu-item index="/rules">
          <el-icon><Guide /></el-icon>
          <template #title>聚合规则</template>
        </el-menu-item>
        <el-menu-item index="/publish">
          <el-icon><Share /></el-icon>
          <template #title>聚合发布</template>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <template #title>系统设置</template>
        </el-menu-item>
      </el-menu>
      <div class="collapse-btn" @click="isCollapsed = !isCollapsed">
        <el-icon :size="18">
          <Fold v-if="!isCollapsed" />
          <Expand v-else />
        </el-icon>
        <transition name="fade">
          <span v-if="!isCollapsed" style="margin-left: 8px; font-size: 13px">收起菜单</span>
        </transition>
      </div>
    </el-aside>
    <el-container>
      <el-header class="top-header">
        <el-breadcrumb separator="/" style="margin-right: auto">
          <el-breadcrumb-item>后台管理系统</el-breadcrumb-item>
        </el-breadcrumb>
        <el-dropdown @command="handleCommand">
          <span class="user-dropdown">
            <el-avatar :size="28" style="background: #409eff; margin-right: 8px">
              <el-icon><UserFilled /></el-icon>
            </el-avatar>
            管理员
            <el-icon style="margin-left: 4px"><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout" divided>
                <el-icon><SwitchButton /></el-icon>退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main style="padding: 24px; background: #f0f2f5">
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
import { useAuthStore } from '../stores/auth'
import { UserFilled, SwitchButton, VideoCamera, Monitor, Calendar, Picture, Guide, Share, Setting, Fold, Expand, ArrowDown } from '@element-plus/icons-vue'
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const isCollapsed = ref(false)
const activeMenu = computed(() => route.path)
function handleCommand(cmd) {
  if (cmd === 'logout') {
    auth.logout()
    router.push('/login')
  }
}
</script>
<style scoped>
.aside-menu {
  background: linear-gradient(180deg, #1d2b3a 0%, #2c3e50 100%);
  transition: width 0.3s;
  display: flex;
  flex-direction: column;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.15);
  z-index: 10;
}
.logo-area {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
  background: rgba(0, 0, 0, 0.1);
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
  color: #909399;
  cursor: pointer;
  background: rgba(0, 0, 0, 0.15);
  transition: all 0.3s;
  overflow: hidden;
  white-space: nowrap;
}
.collapse-btn:hover {
  color: #fff;
  background: rgba(0, 0, 0, 0.25);
}
.top-header {
  background: rgba(255, 255, 255, 0.95);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
  padding: 0 20px;
  height: 60px;
  backdrop-filter: blur(10px);
}
.user-dropdown {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: #606266;
  font-weight: 500;
  transition: color 0.3s;
}
.user-dropdown:hover {
  color: #409eff;
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