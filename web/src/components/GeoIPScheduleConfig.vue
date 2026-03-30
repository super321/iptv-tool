<template>
  <div class="schedule-config">
    <!-- Days interval -->
    <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px">
      <span class="text-secondary" style="font-size: 13px">{{ t('settings_access_control.geoip_schedule_days') }}</span>
      <el-input-number
        v-model="config.days"
        :min="1"
        :max="30"
        :step="1"
        size="small"
        style="width: 120px"
        @change="emitUpdate"
      />
      <span class="text-secondary" style="font-size: 13px">{{ t('settings_access_control.geoip_schedule_days_unit') }}</span>
    </div>

    <!-- Time points -->
    <div style="margin-top: 8px">
      <span class="text-secondary" style="font-size: 13px; display: block; margin-bottom: 6px">{{ t('settings_access_control.geoip_schedule_times') }}</span>
      <div v-for="(time, idx) in config.times" :key="idx" style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px">
        <el-time-picker
          v-model="config.times[idx]"
          format="HH:mm"
          value-format="HH:mm"
          :placeholder="t('settings_access_control.geoip_schedule_time_placeholder')"
          size="small"
          style="width: 140px"
          :clearable="false"
          @change="emitUpdate"
        />
        <el-button v-if="config.times.length > 0" size="small" text type="danger" @click="removeTime(idx)">
          <el-icon><Minus /></el-icon>
        </el-button>
      </div>
      <el-button v-if="config.times.length < 5" size="small" text type="primary" @click="addTime">
        <el-icon><Plus /></el-icon>
        {{ t('settings_access_control.geoip_schedule_add_time') }}
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Minus } from '@element-plus/icons-vue'

const { t } = useI18n()

const props = defineProps({
  /** JSON string of ScheduleConfig (mode=daily), e.g. '{"mode":"daily","days":3,"times":["04:00"]}' */
  modelValue: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])

const config = reactive({
  days: 3,
  times: [],
})

function parseValue(val) {
  if (!val || val === '') {
    config.days = 3
    config.times = []
    return
  }
  try {
    const parsed = JSON.parse(val)
    config.days = parsed.days || 3
    config.times = parsed.times ? [...parsed.times] : []
  } catch {
    config.days = 3
    config.times = []
  }
}

watch(() => props.modelValue, (newVal) => {
  parseValue(newVal)
}, { immediate: true })

function emitUpdate() {
  const obj = { mode: 'daily', days: config.days }
  const validTimes = config.times.filter(t => t)
  if (validTimes.length > 0) {
    obj.times = validTimes
  }
  emit('update:modelValue', JSON.stringify(obj))
}

function addTime() {
  if (config.times.length < 5) {
    config.times.push('')
    // Don't emit here — wait for user to pick a time
  }
}

function removeTime(idx) {
  config.times.splice(idx, 1)
  emitUpdate()
}
</script>
