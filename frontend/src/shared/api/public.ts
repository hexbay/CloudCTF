import { request } from "./http";

export type PublicVersionPayload = {
  service?: string;
  version?: string;
};

export type PublicSettingsPayload = {
  show_github_link?: boolean;
};

export const publicApi = {
  version() {
    return request<PublicVersionPayload>("/api/public/version");
  },
  settings() {
    return request<{ data?: PublicSettingsPayload }>("/api/public/settings");
  },
};
