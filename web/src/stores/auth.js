import { defineStore } from 'pinia'
import api from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    initialized: null, // null = unknown, true/false = checked
  }),
  getters: {
    isLoggedIn: (state) => !!state.token,
  },
  actions: {
    async checkInit() {
      const { data } = await api.get('/init/status')
      this.initialized = data.initialized
      return data.initialized
    },
    async init(username, password) {
      await api.post('/init', { username, password })
      this.initialized = true
    },
    async login(username, password) {
      const { data } = await api.post('/login', { username, password })
      this.token = data.token
      localStorage.setItem('token', data.token)
    },
    logout() {
      this.token = ''
      localStorage.removeItem('token')
    },
    async changePassword(oldPassword, newPassword) {
      await api.post('/user/password', {
        old_password: oldPassword,
        new_password: newPassword,
      })
    },
  },
})
