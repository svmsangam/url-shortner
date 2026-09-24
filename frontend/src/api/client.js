/**
 * API transport boundary: React components -> Axios REST request -> Go API.
 * The interceptor pair carries the device token without React state coupling.
 */
import axios from 'axios';

const TOKEN_KEY = 'device_token_data';
const SEVEN_DAYS_MS = 7 * 24 * 60 * 60 * 1000;

/** Read a non-expired device token, tolerating browsers without localStorage. */
function getStoredToken() {
  try {
    const item = localStorage.getItem(TOKEN_KEY);
    if (!item) return null;
    const { token, timestamp } = JSON.parse(item);
    if (Date.now() - timestamp > SEVEN_DAYS_MS) {
      localStorage.removeItem(TOKEN_KEY);
      return null;
    }
    return token;
  } catch {
    return null; // LocalStorage blocked/incognito fallback
  }
}

/** Persist the device token when storage is available; the API remains stateless otherwise. */
function setStoredToken(token) {
  try {
    localStorage.setItem(TOKEN_KEY, JSON.stringify({ token, timestamp: Date.now() }));
  } catch {
    // Incognito mode prevents writing to localStorage; continue statelessly
  }
}

/** Shared REST client for URL creation, listing, redirect lookup, and analytics. */
const api = axios.create({
  baseURL: 'http://localhost:8080/api', // Point to Go backend port
});

api.interceptors.request.use((config) => {
  const token = getStoredToken();
  if (token) {
    config.headers['X-Device-Token'] = token;
  }
  return config;
});

api.interceptors.response.use((response) => {
  const newToken = response.headers['x-device-token'];
  if (newToken) {
    setStoredToken(newToken);
  }
  return response;
});

export default api;