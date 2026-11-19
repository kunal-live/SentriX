export const API_BASE =
  typeof window !== 'undefined' && window.location.origin && !window.location.origin.includes(':3000')
    ? `${window.location.origin}/api/v1`
    : 'http://localhost:8080/api/v1';

// Secure in-memory access token storage (protects against XSS localStorage theft)
let inMemoryAccessToken: string | null = null;

export function getAccessToken(): string | null {
  return inMemoryAccessToken;
}

export function getRefreshToken(): string | null {
  // Refresh token is maintained securely in an httpOnly SameSite cookie by the browser
  return null;
}

export function setTokens(accessToken: string, _refreshToken?: string): void {
  inMemoryAccessToken = accessToken;

  // Clean up any legacy or deprecated tokens from localStorage to prevent XSS exfiltration
  if (typeof window !== 'undefined' && window.localStorage) {
    try {
      localStorage.removeItem('sentrix_access_token');
      localStorage.removeItem('sentrix_refresh_token');
    } catch {
      // Ignore storage errors in restricted contexts
    }
  }
}

export function clearTokens(): void {
  inMemoryAccessToken = null;

  if (typeof window !== 'undefined' && window.localStorage) {
    try {
      localStorage.removeItem('sentrix_access_token');
      localStorage.removeItem('sentrix_refresh_token');
    } catch {
      // Ignore
    }
  }
}

let activeRefreshPromise: Promise<boolean> | null = null;

export async function refreshSession(): Promise<boolean> {
  if (activeRefreshPromise) {
    return activeRefreshPromise;
  }

  activeRefreshPromise = (async () => {
    try {
      const res = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({}),
      });

      if (!res.ok) {
        clearTokens();
        return false;
      }

      const data = await res.json();
      if (data?.access_token) {
        setTokens(data.access_token, data.refresh_token);
        return true;
      }

      clearTokens();
      return false;
    } catch {
      return false;
    } finally {
      activeRefreshPromise = null;
    }
  })();

  return activeRefreshPromise;
}

export async function publicFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers = new Headers(options.headers);

  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: 'include',
    headers,
  });

  if (!res.ok) {
    throw new Error(`API error: ${res.status}`);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json();
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers = new Headers(options.headers);

  let accessToken = getAccessToken();

  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  let res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: 'include',
    headers,
  });

  if (res.status === 401) {
    const refreshed = await refreshSession();

    if (refreshed) {
      accessToken = getAccessToken();
      if (accessToken) {
        headers.set('Authorization', `Bearer ${accessToken}`);
      }

      res = await fetch(`${API_BASE}${path}`, {
        ...options,
        credentials: 'include',
        headers,
      });
    }
  }

  if (!res.ok) {
    throw new Error(`API error: ${res.status}`);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json();
}

export async function fetchJSON<T>(path: string): Promise<T> {
  return apiFetch<T>(path, {
    method: 'GET',
  });
}

export async function mutateJSON<T>(
  path: string,
  method: 'POST' | 'PATCH' | 'DELETE',
  body?: unknown
): Promise<T> {
  return apiFetch<T>(path, {
    method,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}
