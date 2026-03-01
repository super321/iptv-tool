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
      if (status === 401) {
        // If already on login page or this is a login request, show the actual error message
        const isLoginPage = window.location.hash === '#/login' || window.location.hash === '#/'
        const isLoginRequest = error.config && error.config.url === '/login' && error.config.method === 'post'
        if (isLoginPage || isLoginRequest) {
          ElMessage.error((data && data.error) || '用户名或密码错误')
        } else {
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
