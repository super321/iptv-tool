/**
 * Utility functions for cryptographic operations.
 *
 * Uses Web Crypto API (RSA-OAEP) in secure contexts (HTTPS / localhost).
 * Falls back to JSEncrypt (RSA PKCS#1 v1.5) in insecure contexts (plain HTTP)
 * where crypto.subtle is unavailable.
 */

/**
 * Returns true when the Web Crypto API is available (secure context).
 */
function isWebCryptoAvailable() {
  return !!(window.crypto && window.crypto.subtle)
}

// ─── Web Crypto API helpers (secure context) ───────────────────────────

/**
 * Converts a PEM encoded public key string to an ArrayBuffer.
 */
function pemToArrayBuffer(pem) {
  const b64 = pem
    .replace(/(-----(BEGIN|END) PUBLIC KEY-----|[\n\r])/g, '')
    .trim()

  const binaryString = window.atob(b64)
  const len = binaryString.length
  const bytes = new Uint8Array(len)
  for (let i = 0; i < len; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }
  return bytes.buffer
}

/**
 * Encrypts with Web Crypto API (RSA-OAEP, SHA-256).
 */
async function encryptWithWebCrypto(plaintext, pemKey) {
  const binaryDer = pemToArrayBuffer(pemKey)

  const cryptoKey = await window.crypto.subtle.importKey(
    'spki',
    binaryDer,
    {
      name: 'RSA-OAEP',
      hash: 'SHA-256',
    },
    false,
    ['encrypt']
  )

  const encoder = new TextEncoder()
  const encodedData = encoder.encode(plaintext)

  const encryptedBuffer = await window.crypto.subtle.encrypt(
    { name: 'RSA-OAEP' },
    cryptoKey,
    encodedData
  )

  const encryptedArray = new Uint8Array(encryptedBuffer)
  let binary = ''
  for (let i = 0; i < encryptedArray.byteLength; i++) {
    binary += String.fromCharCode(encryptedArray[i])
  }
  return window.btoa(binary)
}

// ─── JSEncrypt fallback (insecure context) ─────────────────────────────

/**
 * Encrypts with JSEncrypt (RSA PKCS#1 v1.5).
 */
async function encryptWithJSEncrypt(plaintext, pemKey) {
  const { JSEncrypt } = await import('jsencrypt')
  const encrypt = new JSEncrypt()
  encrypt.setPublicKey(pemKey)
  const result = encrypt.encrypt(plaintext)
  if (result === false) {
    throw new Error('JSEncrypt encryption failed')
  }
  return result
}

// ─── Public API ────────────────────────────────────────────────────────

/**
 * Encrypts a plaintext string using RSA and the provided PEM public key.
 * Returns the ciphertext as a Base64 string.
 *
 * Automatically selects the best available encryption method:
 * - Secure context: Web Crypto API (RSA-OAEP)
 * - Insecure context: JSEncrypt (RSA PKCS#1 v1.5)
 *
 * @param {string} plaintext The text to encrypt (e.g., password).
 * @param {string} pemKey The PEM encoded public key.
 * @returns {Promise<string>} Base64 encoded ciphertext.
 */
export async function encryptRSA(plaintext, pemKey) {
  try {
    if (isWebCryptoAvailable()) {
      return await encryptWithWebCrypto(plaintext, pemKey)
    }
    console.warn('Web Crypto API unavailable (insecure context), using JSEncrypt fallback')
    return await encryptWithJSEncrypt(plaintext, pemKey)
  } catch (error) {
    console.error('RSA Encryption failed:', error)
    throw new Error('Encryption failed')
  }
}
