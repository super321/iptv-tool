import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

// Request interceptor: attach JWT token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle errors globally
api.interceptors.response.use(
  response => response,
  error => {
    if (error.response) {
      const { status, data } = error.response
      const isLoginRequest = error.config && error.config.url === '/login' && error.config.method === 'post'

      if (status === 429) {
        // 频率限制 - 登录请求由 Login.vue 自行处理，其他请求全局提示
        if (!isLoginRequest) {
          ElMessage.error((data && data.error) || '请求过于频繁，请稍后再试')
        }
      } else if (status === 403 && isLoginRequest) {
        // 验证码相关的 403 由 Login.vue 自行处理，不在此全局弹出
      } else if (status === 401) {
        if (isLoginRequest) {
          // 登录请求的 401 由 Login.vue 自行处理，不在此全局弹出
        } else {
          // 其他请求的 401 表示 token 过期
          localStorage.removeItem('token')
          window.location.hash = '#/login'
          ElMessage.error('登录已过期，请重新登录')
        }
      } else if (data && data.error) {
        ElMessage.error(data.error)
      }
    } else {
      ElMessage.error('网络请求失败')
    }
    return Promise.reject(error)
  }
)

export default api
