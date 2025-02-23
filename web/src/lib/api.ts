const API_URL = process.env.NEXT_PUBLIC_API_URL;

interface EncryptedContent {
  encrypted: string;
  salt: string;
  iv: string;
}

interface CreateSecretRequest {
  encryptedContent: EncryptedContent;
  customName?: string;
  expiresAt?: string;
  maxViews?: number;
  captchaToken: string;
}

export interface ViewSecretRequest {
  captchaToken: string;
}

export interface ViewSecretResponse {
  encryptedContent: {
    encrypted: string;
    salt: string;
    iv: string;
  };
  expiresAt?: string;
  maxViews?: number;
  viewCount: number;
  error?: string;
  message?: string;
  isBurnAfterReading?: boolean;
}

export async function createSecret(data: CreateSecretRequest) {
  const response = await fetch(`${API_URL}/api/secrets`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || "Failed to create secret");
  }

  return response.json();
}

export async function viewSecret(id: string, request: ViewSecretRequest): Promise<ViewSecretResponse> {
  const response = await fetch(`${API_URL}/api/secrets/${id}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || "Failed to view secret");
  }

  return response.json();
}

export async function viewSecretByName(name: string, request: ViewSecretRequest): Promise<ViewSecretResponse> {
  const response = await fetch(`${API_URL}/api/secrets/name/${name}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || "Failed to view secret");
  }

  return response.json();
} 