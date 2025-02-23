import { Buffer } from 'buffer';

const SALT_LENGTH = 16;
const IV_LENGTH = 12;
const ITERATIONS = 100000;

interface EncryptedData {
  encrypted: string;  // Base64 encoded encrypted data
  salt: string;      // Base64 encoded salt
  iv: string;        // Base64 encoded IV
}

export async function generateSalt(): Promise<Uint8Array> {
  return crypto.getRandomValues(new Uint8Array(SALT_LENGTH));
}

export async function generateIV(): Promise<Uint8Array> {
  return crypto.getRandomValues(new Uint8Array(IV_LENGTH));
}

async function deriveKey(password: string, salt: Uint8Array): Promise<CryptoKey> {
  // Convert password to buffer
  const passwordBuffer = new TextEncoder().encode(password);

  // Import password as raw key material
  const baseKey = await crypto.subtle.importKey(
    "raw",
    passwordBuffer,
    "PBKDF2",
    false,
    ["deriveKey"]
  );

  // Derive AES-GCM key using PBKDF2
  return crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      salt: salt,
      iterations: ITERATIONS,
      hash: "SHA-256",
    },
    baseKey,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"]
  );
}

export async function encrypt(message: string, password: string): Promise<EncryptedData> {
  // Generate salt and IV
  const salt = await generateSalt();
  const iv = await generateIV();

  // Derive key from password
  const key = await deriveKey(password, salt);

  // Encrypt the message
  const encodedMessage = new TextEncoder().encode(message);
  const encryptedData = await crypto.subtle.encrypt(
    {
      name: "AES-GCM",
      iv: iv,
    },
    key,
    encodedMessage
  );

  // Convert to base64
  const encryptedBase64 = Buffer.from(new Uint8Array(encryptedData)).toString('base64');
  const saltBase64 = Buffer.from(salt).toString('base64');
  const ivBase64 = Buffer.from(iv).toString('base64');

  return {
    encrypted: encryptedBase64,
    salt: saltBase64,
    iv: ivBase64,
  };
}

export async function decrypt(
  encryptedData: string,
  salt: string,
  iv: string,
  password: string
): Promise<string> {
  // Convert base64 to buffers
  const encryptedBuffer = Buffer.from(encryptedData, 'base64');
  const saltBuffer = Buffer.from(salt, 'base64');
  const ivBuffer = Buffer.from(iv, 'base64');

  // Derive key from password
  const key = await deriveKey(password, saltBuffer);

  // Decrypt the data
  const decryptedData = await crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv: ivBuffer,
    },
    key,
    encryptedBuffer
  );

  // Convert decrypted data to string
  return new TextDecoder().decode(decryptedData);
} 