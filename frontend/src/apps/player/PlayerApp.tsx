import { useEffect, useState } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { PlayerAuthModal } from './components/PlayerAuthModal';
import { PlayerLayout } from './layouts/PlayerLayout';
import { PlayerAccountPage } from './pages/PlayerAccountPage';
import { PlayerChallengeDetailPage } from './pages/PlayerChallengeDetailPage';
import { PlayerChallengesPage } from './pages/PlayerChallengesPage';
import {
  ArenaOverviewPage,
  PlayerCompetitionArenaPage,
} from './pages/PlayerCompetitionArenaPage';
import { PlayerCompetitionPage } from './pages/PlayerCompetitionPage';
import { PlayerDashboardPage } from './pages/PlayerDashboardPage';
import { PlayerLoginPage } from './pages/PlayerLoginPage';
import { PlayerNotificationsPage } from './pages/PlayerNotificationsPage';
import { PlayerScoreboardPage } from './pages/PlayerScoreboardPage';
import { PlayerTeamPage } from './pages/PlayerTeamPage';
import { PlayerVulnLabDetailPage } from './pages/PlayerVulnLabDetailPage';
import { PlayerVulnLabsPage } from './pages/PlayerVulnLabsPage';
import { CompetitionArenaAnnouncementsPage } from './pages/competition-arena/CompetitionArenaAnnouncementsPage';
import { CompetitionArenaChallengesPage } from './pages/competition-arena/CompetitionArenaChallengesPage';
import { CompetitionArenaRegisterPage } from './pages/competition-arena/CompetitionArenaRegisterPage';
import { CompetitionArenaScoreboardPage } from './pages/competition-arena/CompetitionArenaScoreboardPage';
import { CompetitionArenaWriteupPage } from './pages/competition-arena/CompetitionArenaWriteupPage';

export function PlayerApp() {
  const [authOpen, setAuthOpen] = useState(false);
  const [authTab, setAuthTab] = useState<'login' | 'register'>('login');

  useEffect(() => {
    const listener = (event: Event) => {
      const detail = (event as CustomEvent<'login' | 'register'>).detail;
      setAuthTab(detail === 'register' ? 'register' : 'login');
      setAuthOpen(true);
    };
    window.addEventListener('show-player-auth', listener);
    return () => window.removeEventListener('show-player-auth', listener);
  }, []);

  return (
    <>
      <Routes>
        <Route path="login" element={<PlayerLoginPage />} />
        <Route path="competition-arena" element={<PlayerCompetitionArenaPage />}>
          <Route index element={<Navigate to="overview" replace />} />
          <Route path="overview" element={<ArenaOverviewPage />} />
          <Route path="register" element={<CompetitionArenaRegisterPage />} />
          <Route path="challenges" element={<CompetitionArenaChallengesPage />} />
          <Route path="scoreboard" element={<CompetitionArenaScoreboardPage />} />
          <Route path="writeup" element={<CompetitionArenaWriteupPage />} />
          <Route path="announcements" element={<CompetitionArenaAnnouncementsPage />} />
        </Route>
        <Route element={<PlayerLayout />}>
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<PlayerDashboardPage />} />
          <Route path="competition/*" element={<PlayerCompetitionPage />} />
          <Route path="challenges" element={<PlayerChallengesPage />} />
          <Route path="challenges/:id" element={<PlayerChallengeDetailPage />} />
          <Route path="vuln-labs/:id" element={<PlayerVulnLabDetailPage />} />
          <Route path="vuln-labs/*" element={<PlayerVulnLabsPage />} />
          <Route path="team/*" element={<PlayerTeamPage />} />
          <Route path="scoreboard" element={<PlayerScoreboardPage />} />
          <Route path="notifications" element={<PlayerNotificationsPage />} />
          <Route path="account/*" element={<PlayerAccountPage />} />
        </Route>
      </Routes>
      <PlayerAuthModal open={authOpen} initialTab={authTab} onClose={() => setAuthOpen(false)} />
    </>
  );
}
