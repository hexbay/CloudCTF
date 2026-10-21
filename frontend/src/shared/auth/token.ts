const TOKEN_KEY = "cloudctf_token";
const REFRESH_TOKEN_KEY = "cloudctf_refresh_token";

export type TokenPayload = {
  access_token?: string;
  refresh_token?: string;
  token?: string;
};

export function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

export function setTokens(payload: TokenPayload) {
  const accessToken = payload.access_token ?? payload.token;
  if (accessToken) {
    localStorage.setItem(TOKEN_KEY, accessToken);
    localStorage.removeItem("cloudctf_admin_token");
  }
  if (payload.refresh_token) {
    localStorage.setItem(REFRESH_TOKEN_KEY, payload.refresh_token);
    localStorage.removeItem("cloudctf_admin_refresh_token");
  }
}

export function clearTokens() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem("cloudctf_admin_token");
  localStorage.removeItem("cloudctf_admin_refresh_token");
}

export const getAdminToken = getToken;
export const setAdminTokens = setTokens;
export const clearAdminTokens = clearTokens;
