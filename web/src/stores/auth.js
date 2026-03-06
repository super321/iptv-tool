import { defineStore } from 'pinia'
import api from '../api'
import { encryptRSA } from '../utils/crypto'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    initialized: null, // null = unknown, true/false = checked
    publicKey: null,
  }),
  getters: {
    isLoggedIn: (state) => !!state.token,
  },
  actions: {
    async fetchPublicKey() {
      if (!this.publicKey) {
        const { data } = await api.get('/system/pubkey')
        this.publicKey = data
      }
      return this.publicKey
    },
    async checkInit() {
      const { data } = await api.get('/init/status')
      this.initialized = data.initialized
      return data.initialized
    },
    async init(username, password) {
      const pubKey = await this.fetchPublicKey()
      const encryptedPassword = await encryptRSA(password, pubKey)
      await api.post('/init', { username, password: encryptedPassword })
      this.initialized = true
    },
    async login(username, password, captchaId, captchaCode) {
      const pubKey = await this.fetchPublicKey()
      const encryptedPassword = await encryptRSA(password, pubKey)
      
      const payload = { username, password: encryptedPassword }
      if (captchaId) {
        payload.captcha_id = captchaId
        payload.captcha_code = captchaCode
      }
      const { data } = await api.post('/login', payload)
      this.token = data.token
      localStorage.setItem('token', data.token)
    },
    logout() {
      this.token = ''
      localStorage.removeItem('token')
    },
    async getCaptcha() {
      const { data } = await api.get('/captcha')
      return data // { captcha_id, captcha_image }
    },
    async changePassword(oldPassword, newPassword) {
      const pubKey = await this.fetchPublicKey()
      const encryptedOldPassword = await encryptRSA(oldPassword, pubKey)
      const encryptedNewPassword = await encryptRSA(newPassword, pubKey)
      
      await api.post('/user/password', {
        old_password: encryptedOldPassword,
        new_password: encryptedNewPassword,
      })
    },
  },
})
