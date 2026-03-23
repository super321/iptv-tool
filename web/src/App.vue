<template>
  <el-config-provider :locale="elementLocale">
    <router-view />
  </el-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import zhTw from 'element-plus/dist/locale/zh-tw.mjs'
import en from 'element-plus/dist/locale/en.mjs'

const { locale } = useI18n()
const localeMap = { zh: zhCn, 'zh-Hant': zhTw, en: en }
const elementLocale = computed(() => localeMap[locale.value] || en)
</script>
