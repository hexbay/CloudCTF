export type PlayerUser = {
  id?: number;
  username?: string;
  email?: string;
  full_name?: string;
  team_id?: number;
  team_name?: string;
  role?: string;
  roles?: string[];
  permissions?: string[];
  is_admin?: boolean;
  is_staff?: boolean;
};

export function getStoredPlayerUser(): PlayerUser | null {
  const raw = localStorage.getItem("cloudctf_user");
  if (!raw) return null;
  try {
    return JSON.parse(raw) as PlayerUser;
  } catch {
    return null;
  }
}

export function isPlayerLoggedIn() {
  return Boolean(localStorage.getItem("cloudctf_token"));
}

export function clearPlayerAuth() {
  localStorage.removeItem("cloudctf_token");
  localStorage.removeItem("cloudctf_refresh_token");
  localStorage.removeItem("cloudctf_user");
}
