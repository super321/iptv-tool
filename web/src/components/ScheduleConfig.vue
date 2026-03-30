<template>
  <div class="schedule-config">
    <!-- Empty state / toggle -->
    <div v-if="!hasSchedule" style="display: flex; align-items: center; gap: 8px">
      <el-button size="small" @click="enableSchedule">
        <el-icon><Plus /></el-icon>
        {{ enableLabel || t(i18nPrefix + '.schedule_mode') }}
      </el-button>
    </div>

    <div v-else>
      <!-- Mode selector + clear -->
      <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px">
        <el-radio-group v-model="config.mode" size="small" @change="onModeChange">
          <el-radio-button value="interval">{{ t(i18nPrefix + '.schedule_mode_interval') }}</el-radio-button>
          <el-radio-button value="daily">{{ t(i18nPrefix + '.schedule_mode_daily') }}</el-radio-button>
        </el-radio-group>
        <el-button size="small" text type="danger" @click="clearSchedule">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>

      <!-- Interval mode -->
      <div v-if="config.mode === 'interval'" style="display: flex; align-items: center; gap: 8px">
        <el-input-number
          v-model="config.hours"
          :min="1"
          :max="48"
          :step="1"
          size="small"
          style="width: 140px"
          @change="emitUpdate"
        />
        <span class="text-secondary" style="font-size: 13px">{{ t(i18nPrefix + '.schedule_hours_unit') }}</span>
      </div>

      <!-- Daily mode -->
      <div v-if="config.mode === 'daily'">
        <div v-for="(time, idx) in config.times" :key="idx" style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px">
          <el-time-picker
            v-model="config.times[idx]"
            format="HH:mm"
            value-format="HH:mm"
            :placeholder="t(i18nPrefix + '.schedule_time_placeholder')"
            size="small"
            style="width: 140px"
            :clearable="false"
            @change="emitUpdate"
          />
          <el-button v-if="config.times.length > 1" size="small" text type="danger" @click="removeTime(idx)">
            <el-icon><Minus /></el-icon>
          </el-button>
        </div>
        <el-button v-if="config.times.length < 5" size="small" text type="primary" @click="addTime">
          <el-icon><Plus /></el-icon>
          {{ t(i18nPrefix + '.schedule_add_time') }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Close, Minus } from '@element-plus/icons-vue'

const { t } = useI18n()

const props = defineProps({
  /** JSON string of ScheduleConfig, e.g. '{"mode":"interval","hours":6}' or '' */
  modelValue: { type: String, default: '' },
  /** i18n prefix: 'live_sources' or 'epg_sources' */
  i18nPrefix: { type: String, default: 'live_sources' },
  /** Label for the enable button */
  enableLabel: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])

const config = reactive({
  mode: '',
  hours: 6,
  times: [],
})

const hasSchedule = computed(() => {
  return config.mode !== ''
})

// Parse incoming modelValue
function parseValue(val) {
  if (!val || val === '') {
    config.mode = ''
    config.hours = 6
    config.times = []
    return
  }
  try {
    const parsed = JSON.parse(val)
    config.mode = parsed.mode || 'interval'
    config.hours = parsed.hours || 6
    config.times = (parsed.times && parsed.times.length > 0) ? [...parsed.times] : []
  } catch {
    config.mode = ''
    config.hours = 6
    config.times = []
  }
}

// Watch for external changes
watch(() => props.modelValue, (newVal) => {
  parseValue(newVal)
}, { immediate: true })

function emitUpdate() {
  if (config.mode === '') {
    emit('update:modelValue', '')
    return
  }
  const obj = { mode: config.mode }
  if (config.mode === 'interval') {
    obj.hours = config.hours
  } else if (config.mode === 'daily') {
    obj.times = config.times.filter(t => t)
  }
  emit('update:modelValue', JSON.stringify(obj))
}

function enableSchedule() {
  config.mode = 'interval'
  config.hours = 6
  config.times = []
  emitUpdate()
}

function clearSchedule() {
  config.mode = ''
  config.hours = 6
  config.times = []
  emitUpdate()
}

function onModeChange() {
  // When switching to daily mode, add one empty time slot if none exist
  if (config.mode === 'daily' && config.times.length === 0) {
    config.times.push('')
  }
  emitUpdate()
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
