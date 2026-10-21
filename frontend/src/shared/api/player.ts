import { request } from './http';

export type ListResponse<T> = {
  data?: T[];
  success?: boolean;
  total?: number;
  current?: number;
  pageSize?: number;
};

export type PlayerCompetition = {
  id: number;
  name: string;
  type?: string;
  status: string;
  start_time: string;
  end_time: string;
  is_public?: boolean;
  banner_url?: string;
  max_team_members?: number;
  description?: string;
  participant_mode?: string;
  registration_open?: boolean;
  is_participant?: boolean;
  challenge_count?: number;
  announcement_count?: number;
};

export type PlayerAnnouncement = {
  id: number;
  title: string;
  content?: string;
  summary?: string;
  is_pinned?: boolean;
  is_published?: boolean;
  target_type?: string;
  target_id?: number;
  published_at?: string;
  created_at?: string;
};

export type PlayerChallenge = {
  id: number;
  name: string;
  description?: string;
  category: string;
  difficulty: number | string;
  points: number;
  is_solved?: boolean;
  has_level_one_solution?: boolean;
};

export type PlayerChallengeHint = {
  index: number;
  content: string;
  cost: number;
  purchased: boolean;
};

export type PlayerChallengeContainerTarget = {
  port: number;
  protocol: string;
  container_port: number;
  url: string;
};

export type PlayerChallengeContainerInfo = {
  container_id: string;
  status: string;
  created_at?: string;
  updated_at?: string;
  expires_at?: string;
  expires?: number;
  name?: string;
  targets: PlayerChallengeContainerTarget[];
};

export type PlayerChallengeDetail = PlayerChallenge & {
  solved?: boolean;
  solves_count?: number;
  bloods?: Array<{ rank: number; user_id: number; username: string; submitted_at: string }>;
  hints?: PlayerChallengeHint[];
  attachments?: Array<{ filename?: string; file_path?: string; url?: string }>;
  solved_at?: string;
  user_points?: number;
  is_dynamic?: boolean;
  container_image?: string;
  container_port?: number;
  container_timeout?: number;
  container_info?: PlayerChallengeContainerInfo | null;
};

export type PlayerExamQuestion = {
  id: number;
  question: string;
  question_type: 'single' | 'multiple' | 'judge' | 'text';
  points: number;
  context?: {
    options?: Array<{ text: string; checked?: boolean }>;
  };
};

export type PlayerExam = {
  exam_id: number;
  title: string;
  total_score: number;
  competition_id?: number;
  questions: PlayerExamQuestion[];
  submission?: {
    correct?: number;
    total?: number;
    score?: number;
    total_score?: number;
  } | null;
};

export type PlayerStats = {
  total_points?: number;
  solved_challenges?: number;
  rank?: number;
  team_name?: string;
};

export type PublicSystemSettings = {
  site_name?: string;
  site_title?: string;
  site_logo?: string;
  site_description?: string;
  allow_registration?: boolean;
  enable_notifications?: boolean;
  enable_writeup?: boolean;
  max_team_size?: number;
};

export type PlayerRank = {
  rank: number;
  user_id: number;
  username: string;
  solved_count: number;
  total_points: number;
};

export type PlayerWriteup = {
  id: number;
  competition_id: number;
  user_id: number;
  team_id?: number | null;
  title: string;
  file_name: string;
  filename?: string;
  file_size: number;
  size?: number;
  url?: string;
  created_at: string;
  updated_at: string;
};

export type ChallengeRank = {
  rank: number;
  challenge_id: number;
  challenge_name: string;
  category: string;
  difficulty: number;
  points: number;
  solved_count: number;
  total_submissions: number;
};

export type PlayerTeam = {
  id: number;
  name: string;
  description?: string;
  invitation_code?: string;
  created_at?: string;
  member_count?: number;
  members?: Array<{
    id: number;
    username: string;
    email?: string;
    full_name?: string;
    is_owner?: boolean;
  }>;
  competitions?: Array<{ id: number; name: string; status: string }>;
};

export type VulnLab = {
  id: number;
  name: string;
  cve_id?: string;
  severity?: string;
  summary?: string;
  description?: string;
  reproduce_steps?: string;
  reference_links?: string[];
  template_id?: number;
  timeout_seconds?: number;
  is_active?: boolean;
  is_public?: boolean;
  runtime?: PlayerVulnLabRuntime | null;
  created_at?: string;
  updated_at?: string;
};

export type PlayerVulnLabTarget = {
  port?: number | string;
  protocol?: string;
  container_port?: number | string;
  url: string;
};

export type PlayerVulnLabRuntime = {
  group_id?: number;
  container_id?: string;
  status?: string;
  created_at?: string;
  updated_at?: string;
  expires_at?: string;
  expires?: number;
  name?: string;
  targets?: PlayerVulnLabTarget[];
};

export type PlayerVulnLabDetail = VulnLab & {
  runtime?: PlayerVulnLabRuntime | null;
};

export type PlayerNotification = {
  id: number;
  title: string;
  content?: string;
  message_type?: string;
  audience?: 'player';
  priority?: string;
  is_read?: boolean;
  created_at?: string;
};

export type PlayerUser = {
  id: number;
  username: string;
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

export type PlayerLoginResponse = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_at: string;
  user?: PlayerUser;
};

export type PlayerRegisterPayload = {
  username: string;
  email: string;
  password: string;
  full_name?: string;
};

export type PlayerRegisterResponse = {
  success?: boolean;
  data?: PlayerUser;
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_at: string;
};

export const playerApi = {
  login(data: { username: string; password: string }) {
    return request<PlayerLoginResponse>('/api/login', {
      method: 'POST',
      body: data,
    });
  },
  register(data: PlayerRegisterPayload) {
    return request<PlayerRegisterResponse>('/api/register', {
      method: 'POST',
      body: data,
    });
  },
  me() {
    return request<PlayerUser>('/api/me');
  },
  competitions(query?: Record<string, string | number | boolean | undefined>) {
    return request<ListResponse<PlayerCompetition>>('/api/competitions', { query });
  },
  competitionDetail(id: number) {
    return request<PlayerCompetition>(`/api/competitions/${id}`);
  },
  competitionAnnouncements(id: number, query?: Record<string, string | number | boolean | undefined>) {
    return request<ListResponse<PlayerAnnouncement>>(`/api/competitions/${id}/announcements`, { query });
  },
  competitionScoreboard(id: number) {
    return request<ListResponse<PlayerRank>>(`/api/competitions/${id}/scoreboard`);
  },
  competitionExam(id: number) {
    return request<PlayerExam>(`/api/competitions/${id}/exam`);
  },
  competitionWriteup(id: number) {
    return request<PlayerWriteup | { data: null; message?: string }>(`/api/competitions/${id}/writeup`);
  },
  uploadCompetitionWriteup(id: number, data: FormData) {
    return request<PlayerWriteup>(`/api/competitions/${id}/writeup`, {
      method: 'POST',
      body: data,
    });
  },
  submitCompetitionExam(id: number, data: { exam_id: number; answers: Record<string, unknown> }) {
    return request<{ success?: boolean; correct?: number; total?: number; score?: number; total_score?: number }>(
      `/api/exam/${id}/submit`,
      {
        method: 'POST',
        body: data,
      },
    );
  },
  challenges(query?: Record<string, string | number | boolean | undefined>) {
    return request<ListResponse<PlayerChallenge>>('/api/practice/challenges', { query });
  },
  registerCompetition(id: number, data?: { team_id?: number }) {
    return request<{ success?: boolean; message?: string }>(`/api/competitions/${id}/register`, {
      method: 'POST',
      body: data ?? {},
    });
  },
  challengeDetail(id: number) {
    return request<PlayerChallengeDetail>(`/api/challenges/${id}`);
  },
  submitFlag(id: number, flag: string) {
    return request<{ correct?: boolean; points?: number; message?: string; already_solved?: boolean; is_first_blood?: boolean }>(
      `/api/challenges/${id}/submit`,
      { method: 'POST', body: { flag } },
    );
  },
  startChallenge(id: number) {
    return request<{ success?: boolean; container_info?: PlayerChallengeContainerInfo }>(`/api/challenges/${id}/start`, {
      method: 'POST',
    });
  },
  extendChallenge(id: number) {
    return request<{ success?: boolean; message?: string; container_info?: PlayerChallengeContainerInfo }>(
      `/api/challenges/${id}/extend`,
      { method: 'POST' },
    );
  },
  stopChallenge(id: number) {
    return request<{ success?: boolean; container_status?: string }>(`/api/challenges/${id}/stop`, {
      method: 'POST',
    });
  },
  purchaseHint(id: number, hintIndex: number) {
    return request<{ success?: boolean; message?: string; hint?: PlayerChallengeHint }>(
      `/api/challenges/${id}/hints/${hintIndex}/purchase`,
      {
        method: 'POST',
      },
    );
  },
  stats() {
    return request<PlayerStats>('/api/user/stats');
  },
  publicSettings() {
    return request<{ data?: PublicSystemSettings }>('/api/public/settings');
  },
  practiceLeaderboard(query?: Record<string, string | number | boolean | undefined>) {
    return request<ListResponse<PlayerRank>>('/api/leaderboard/practice', { query });
  },
  challengeLeaderboard(query?: Record<string, string | number | boolean | undefined>) {
    return request<ListResponse<ChallengeRank>>('/api/leaderboard/challenges', { query });
  },
  myRank() {
    return request<PlayerRank>('/api/leaderboard/my-rank', { query: { type: 'practice' } });
  },
  team() {
    return request<PlayerTeam>('/api/team/my');
  },
  teams(query?: Record<string, string | number | boolean | undefined>) {
    return request<ListResponse<PlayerTeam>>('/api/teams', { query });
  },
  createTeam(data: { name: string; description?: string }) {
    return request<PlayerTeam>('/api/team', { method: 'POST', body: data });
  },
  joinTeam(data: { invitation_code?: string; team_id?: number }) {
    return request<{ success?: boolean; team_id?: number; team_name?: string }>('/api/team/join', {
      method: 'POST',
      body: data,
    });
  },
  leaveTeam() {
    return request('/api/team/leave', { method: 'POST' });
  },
  notifications(query?: Record<string, string | number | boolean | undefined>) {
    return request<ListResponse<PlayerNotification>>('/api/notifications', { query });
  },
  markAllNotificationsRead() {
    return request<{ success?: boolean }>('/api/notifications/mark-all-read', { method: 'PUT' });
  },
  markNotificationRead(id: number) {
    return request<{ success?: boolean }>(`/api/notifications/${id}/read`, { method: 'PUT' });
  },
  vulnLabs(query?: Record<string, string | number | boolean | undefined>) {
    return request<ListResponse<VulnLab>>('/api/vuln-labs', { query });
  },
  vulnLabDetail(id: number) {
    return request<PlayerVulnLabDetail>(`/api/vuln-labs/${id}`);
  },
  startVulnLab(id: number) {
    return request<{
      success?: boolean;
      lab_id?: number;
      user_id?: number;
      group?: PlayerVulnLabRuntime;
      container_status?: string;
      expires_at?: string;
      message?: string;
    }>(`/api/vuln-labs/${id}/start`, {
      method: 'POST',
    });
  },
  extendVulnLab(id: number) {
    return request<{ success?: boolean; message?: string; container_info?: PlayerVulnLabRuntime }>(
      `/api/vuln-labs/${id}/extend`,
      {
        method: 'POST',
      },
    );
  },
  stopVulnLab(id: number) {
    return request<{ success?: boolean; container_status?: string }>(`/api/vuln-labs/${id}/stop`, {
      method: 'POST',
    });
  },
};
