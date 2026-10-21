import type {
  ChallengeRank,
  PlayerChallenge,
  PlayerCompetition,
  PlayerRank,
  PlayerTeam,
  VulnLab,
} from "@shared/api/player";

export const fallbackCompetitions: PlayerCompetition[] = [
  {
    id: 1,
    name: "Ocean CTF 春季挑战赛",
    type: "ctf",
    status: "ongoing",
    start_time: "2026-05-20T09:00:00+08:00",
    end_time: "2026-06-20T18:00:00+08:00",
    is_public: true,
    max_team_members: 4,
    description: "面向 Web、Pwn、Crypto、Reverse 的综合攻防赛事。",
    participant_mode: "团队赛",
    registration_open: true,
  },
  {
    id: 2,
    name: "Web 安全专项训练营",
    type: "mixed",
    status: "upcoming",
    start_time: "2026-06-10T09:00:00+08:00",
    end_time: "2026-06-18T18:00:00+08:00",
    is_public: true,
    max_team_members: 3,
    description: "结合练兵与正式比赛的专题训练场景。",
    participant_mode: "个人 / 团队",
    registration_open: true,
  },
  {
    id: 3,
    name: "企业攻防演练赛",
    type: "awd",
    status: "finished",
    start_time: "2026-05-01T09:00:00+08:00",
    end_time: "2026-05-03T18:00:00+08:00",
    is_public: true,
    max_team_members: 5,
    description: "偏实战化的攻防演练赛事。",
    participant_mode: "团队赛",
    registration_open: false,
  },
];

export const fallbackUserRanks: PlayerRank[] = [
  {
    rank: 1,
    user_id: 1,
    username: "nightfall",
    solved_count: 36,
    total_points: 7800,
  },
  {
    rank: 2,
    user_id: 2,
    username: "kernel",
    solved_count: 31,
    total_points: 6900,
  },
  {
    rank: 3,
    user_id: 3,
    username: "cipher",
    solved_count: 28,
    total_points: 6240,
  },
  {
    rank: 4,
    user_id: 4,
    username: "cloudctf",
    solved_count: 23,
    total_points: 5100,
  },
  {
    rank: 5,
    user_id: 5,
    username: "rookie",
    solved_count: 18,
    total_points: 3900,
  },
];

export const fallbackChallengeRanks: ChallengeRank[] = [
  {
    rank: 1,
    challenge_id: 101,
    challenge_name: "Easy SQL",
    category: "Web",
    difficulty: 1,
    points: 100,
    solved_count: 96,
    total_submissions: 211,
  },
  {
    rank: 2,
    challenge_id: 102,
    challenge_name: "Baby RSA",
    category: "Crypto",
    difficulty: 2,
    points: 150,
    solved_count: 74,
    total_submissions: 160,
  },
  {
    rank: 3,
    challenge_id: 106,
    challenge_name: "Blind SSRF",
    category: "Web",
    difficulty: 4,
    points: 400,
    solved_count: 18,
    total_submissions: 97,
  },
];

export const fallbackTeam: PlayerTeam = {
  id: 1,
  name: "CloudCTF Team",
  description: "专注 Web、Crypto 与攻防实战的训练队伍。",
  created_at: "2026-05-01T09:00:00+08:00",
  member_count: 3,
  members: [
    {
      id: 1,
      username: "captain",
      email: "captain@example.com",
      is_owner: true,
    },
    { id: 2, username: "webster", email: "web@example.com" },
    { id: 3, username: "cryptoer", email: "crypto@example.com" },
  ],
  competitions: [{ id: 1, name: "Ocean CTF 春季挑战赛", status: "ongoing" }],
};

export const fallbackVulnLabs: VulnLab[] = [
  {
    id: 201,
    name: "Apache Struts2 RCE 复现",
    cve_id: "CVE-2017-5638",
    severity: "CRITICAL",
    summary: "通过 Content-Type 触发 OGNL 表达式执行。",
  },
  {
    id: 202,
    name: "Fastjson 反序列化",
    cve_id: "CVE-2022-25845",
    severity: "HIGH",
    summary: "学习 autoType 风险和 JNDI 利用链。",
  },
  {
    id: 203,
    name: "Nginx 配置错误实验",
    severity: "MEDIUM",
    summary: "复现 alias 配置不当导致的路径穿越。",
  },
  {
    id: 204,
    name: "弱口令服务接管",
    severity: "LOW",
    summary: "理解弱口令风险和基础加固方式。",
  },
];
