import { ref, onUnmounted } from 'vue'

/**
 * Composable for managing polling intervals with automatic cleanup.
 *
 * @param {Function} fn - Async function to call on each interval tick
 * @param {number} interval - Polling interval in milliseconds
 * @returns {{ isPolling, startPolling, stopPolling }}
 *
 * @example
 * const { isPolling, startPolling, stopPolling } = usePolling(
 *   () => loadSources(false),
 *   3000
 * )
 * // Start polling when needed
 * if (hasSyncing) startPolling()
 * // Stop when done — also auto-stops on component unmount
 */
export function usePolling(fn, interval) {
  const isPolling = ref(false)
  let timer = null

  function startPolling() {
    if (timer) return // Already polling
    isPolling.value = true
    timer = setInterval(fn, interval)
  }

  function stopPolling() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    isPolling.value = false
  }

  onUnmounted(stopPolling)

  return { isPolling, startPolling, stopPolling }
}
