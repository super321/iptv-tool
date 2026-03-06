/**
 * Utility functions for cryptographic operations.
 * Uses Web Crypto API for zero-dependency RSA-OAEP encryption.
 */

/**
 * Converts a PEM encoded public key string to an ArrayBuffer.
 * @param {string} pem The PEM encoded public key.
 * @returns {ArrayBuffer}
 */
function str2ab(pem) {
  // Remove the header, footer, and newlines from the PEM string
  const b64 = pem
    .replace(/(-----(BEGIN|END) PUBLIC KEY-----|[\n\r])/g, '')
    .trim()
  
  // Decode base64
  const binaryString = window.atob(b64)
  const len = binaryString.length
  const bytes = new Uint8Array(len)
  for (let i = 0; i < len; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }
  return bytes.buffer
}

/**
 * Encrypts a plaintext string using RSA-OAEP and the provided PEM public key.
 * Returns the ciphertext as a Base64 string.
 *
 * @param {string} plaintext The text to encrypt (e.g., password).
 * @param {string} pemKey The PEM encoded public key.
 * @returns {Promise<string>} Base64 encoded ciphertext.
 */
export async function encryptRSA(plaintext, pemKey) {
  try {
    const binaryDer = str2ab(pemKey)

    // Import the public key
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

    // Encode the plaintext
    const encoder = new TextEncoder()
    const encodedData = encoder.encode(plaintext)

    // Encrypt
    const encryptedBuffer = await window.crypto.subtle.encrypt(
      {
        name: 'RSA-OAEP',
      },
      cryptoKey,
      encodedData
    )

    // Convert encrypted ArrayBuffer to Base64
    const encryptedArray = new Uint8Array(encryptedBuffer)
    let binary = ''
    for (let i = 0; i < encryptedArray.byteLength; i++) {
      binary += String.fromCharCode(encryptedArray[i])
    }
    return window.btoa(binary)
  } catch (error) {
    console.error('RSA Encryption failed:', error)
    throw new Error('Encryption failed')
  }
}
