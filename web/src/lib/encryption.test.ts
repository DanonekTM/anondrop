import { describe, expect, it } from "@jest/globals";
import { encrypt, decrypt } from "./encryption";

describe("Encryption Module", () => {
  const testMessage = "Hello, World!";
  const testPassword = "test-password-123";

  it("should encrypt and decrypt a message successfully", async () => {
    // Encrypt the message
    const encryptedData = await encrypt(testMessage, testPassword);

    // Verify encrypted data format
    expect(encryptedData).toHaveProperty("encrypted");
    expect(encryptedData).toHaveProperty("salt");
    expect(encryptedData).toHaveProperty("iv");
    expect(typeof encryptedData.encrypted).toBe("string");
    expect(typeof encryptedData.salt).toBe("string");
    expect(typeof encryptedData.iv).toBe("string");

    // Decrypt the message
    const decryptedMessage = await decrypt(
      encryptedData.encrypted,
      encryptedData.salt,
      encryptedData.iv,
      testPassword
    );

    // Verify the decrypted message matches the original
    expect(decryptedMessage).toBe(testMessage);
  });

  it("should fail to decrypt with wrong password", async () => {
    // Encrypt with correct password
    const encryptedData = await encrypt(testMessage, testPassword);

    // Try to decrypt with wrong password
    await expect(
      decrypt(
        encryptedData.encrypted,
        encryptedData.salt,
        encryptedData.iv,
        "wrong-password"
      )
    ).rejects.toThrow();
  });

  it("should handle empty messages", async () => {
    const emptyMessage = "";
    const encryptedData = await encrypt(emptyMessage, testPassword);
    const decryptedMessage = await decrypt(
      encryptedData.encrypted,
      encryptedData.salt,
      encryptedData.iv,
      testPassword
    );
    expect(decryptedMessage).toBe(emptyMessage);
  });

  it("should handle special characters", async () => {
    const specialMessage = "!@#$%^&*()_+-=[]{}|;:,.<>?";
    const encryptedData = await encrypt(specialMessage, testPassword);
    const decryptedMessage = await decrypt(
      encryptedData.encrypted,
      encryptedData.salt,
      encryptedData.iv,
      testPassword
    );
    expect(decryptedMessage).toBe(specialMessage);
  });

  it("should handle Unicode characters", async () => {
    const unicodeMessage = "Hello, 世界! Привет, мир! 👋";
    const encryptedData = await encrypt(unicodeMessage, testPassword);
    const decryptedMessage = await decrypt(
      encryptedData.encrypted,
      encryptedData.salt,
      encryptedData.iv,
      testPassword
    );
    expect(decryptedMessage).toBe(unicodeMessage);
  });

  it("should generate different encrypted data for same input", async () => {
    const encryptedData1 = await encrypt(testMessage, testPassword);
    const encryptedData2 = await encrypt(testMessage, testPassword);

    // Verify that the encrypted data, salt, and IV are different
    expect(encryptedData1.encrypted).not.toBe(encryptedData2.encrypted);
    expect(encryptedData1.salt).not.toBe(encryptedData2.salt);
    expect(encryptedData1.iv).not.toBe(encryptedData2.iv);
  });
}); 