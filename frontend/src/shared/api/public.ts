import { request } from "./http";

export type PublicVersionPayload = {
  service?: string;
  version?: string;
};

export const publicApi = {
  version() {
    return request<PublicVersionPayload>("/api/public/version");
  },
};
