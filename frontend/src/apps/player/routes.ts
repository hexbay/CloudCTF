export type PlayerRouteItem = {
  path: string;
  title: string;
  legacyPath: string;
};

export const playerRoutes: PlayerRouteItem[] = [
  { path: 'dashboard', title: '首页', legacyPath: '/dashboard' },
  { path: 'competition/*', title: '赛事', legacyPath: '/competition' },
  { path: 'competition-arena/*', title: '比赛场景', legacyPath: '/competition-arena' },
  { path: 'challenges/*', title: '练习题目', legacyPath: '/challenges' },
  { path: 'vuln-labs/*', title: '漏洞复现', legacyPath: '/vuln-labs' },
  { path: 'team/*', title: '队伍', legacyPath: '/team' },
  { path: 'scoreboard', title: '排行榜', legacyPath: '/scoreboard' },
  { path: 'notifications', title: '通知中心', legacyPath: '/notifications' },
  { path: 'account/*', title: '个人中心', legacyPath: '/account' }
];

