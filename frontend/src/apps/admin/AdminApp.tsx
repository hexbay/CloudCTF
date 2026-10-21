import { Navigate, Route, Routes } from 'react-router-dom';
import { AdminLayout } from './layouts/AdminLayout';
import { AdminDashboardPage } from './pages/AdminDashboardPage';
import { AdminLoginPage } from './pages/AdminLoginPage';
import { AdminCompetitionChallengesPage } from './pages/AdminCompetitionChallengesPage';
import { AdminCompetitionDetailPage } from './pages/AdminCompetitionDetailPage';
import { AdminCompetitionExamDetailPage } from './pages/AdminCompetitionExamDetailPage';
import { AdminCompetitionListPage } from './pages/AdminCompetitionListPage';
import { AdminChallengePage } from './pages/AdminChallengePage';
import { AdminAnnouncementsPage } from './pages/AdminAnnouncementsPage';
import { AdminAuditPage } from './pages/AdminAuditPage';
import { AdminExamDetailPage } from './pages/AdminExamDetailPage';
import { AdminExamPage } from './pages/AdminExamPage';
import { AdminNotificationsPage } from './pages/AdminNotificationsPage';
import { AdminSettingsPage } from './pages/AdminSettingsPage';
import { AdminTeamsPage } from './pages/AdminTeamsPage';
import { AdminTheoryPage } from './pages/AdminTheoryPage';
import { AdminUsersPage } from './pages/AdminUsersPage';
import { AdminVulnLabsPage } from './pages/AdminVulnLabsPage';
import { AdminAdminsPage } from './pages/AdminAdminsPage';
import { AdminPracticeSubmissionsPage } from './pages/AdminPracticeSubmissionsPage';
import {
  AdminVirtualizationContainersPage,
  AdminVirtualizationGroupsPage,
  AdminVirtualizationHostContainersPage,
  AdminVirtualizationHostDetailPage,
  AdminVirtualizationHostImagesPage,
  AdminVirtualizationHostsPage,
  AdminVirtualizationTemplatesPage,
} from './pages/AdminVirtualizationSectionPages';
import { AdminRolesPage } from './pages/AdminRolesPage';

export function AdminApp() {
  return (
    <Routes>
      <Route path="login" element={<AdminLoginPage />} />
      <Route element={<AdminLayout />}>
        <Route index element={<Navigate to="dashboard" replace />} />
        <Route path="dashboard" element={<AdminDashboardPage />} />
        <Route path="settings" element={<AdminSettingsPage />} />
        <Route path="settings/:tabKey" element={<AdminSettingsPage />} />
        <Route path="competition/*" element={<AdminCompetitionListPage />} />
        <Route path="competition/:competitionId" element={<AdminCompetitionDetailPage />} />
        <Route path="competition/:competitionId/exams/:examId" element={<AdminCompetitionExamDetailPage />} />
        <Route path="competition/:competitionId/:tabKey" element={<AdminCompetitionDetailPage />} />
        <Route path="users/*" element={<AdminUsersPage />} />
        <Route path="teams/*" element={<AdminTeamsPage />} />
        <Route path="admins" element={<AdminAdminsPage />} />
        <Route path="roles" element={<AdminRolesPage />} />
        <Route path="challenge/*" element={<AdminChallengePage />} />
        <Route path="vuln-lab/*" element={<AdminVulnLabsPage />} />
        <Route path="theory/*" element={<AdminTheoryPage />} />
        <Route path="theory-exams" element={<AdminExamPage />} />
        <Route path="theory-exams/:examId" element={<AdminExamDetailPage />} />
        <Route path="announcements/*" element={<AdminAnnouncementsPage />} />
        <Route path="audit" element={<AdminAuditPage />} />
        <Route path="notifications" element={<AdminNotificationsPage />} />
        <Route path="virtualization/*" element={<AdminVirtualizationHostsPage />} />
        <Route path="virtualization-containers" element={<AdminVirtualizationContainersPage />} />
        <Route path="virtualization-groups" element={<AdminVirtualizationGroupsPage />} />
        <Route path="virtualization-templates" element={<AdminVirtualizationTemplatesPage />} />
        <Route path="resource-repositories" element={<Navigate to="/admin/virtualization-templates" replace />} />
        <Route path="virtualization/:hostId" element={<AdminVirtualizationHostDetailPage />} />
        <Route path="virtualization/:hostId/images" element={<AdminVirtualizationHostImagesPage />} />
        <Route path="virtualization/:hostId/containers" element={<AdminVirtualizationHostContainersPage />} />
        <Route path="competition-environments" element={<Navigate to="/admin/competition" replace />} />
        <Route path="competition-scoreboard" element={<Navigate to="/admin/competition" replace />} />
        <Route path="competition-submissions" element={<Navigate to="/admin/competition" replace />} />
        <Route path="practice-submissions" element={<AdminPracticeSubmissionsPage />} />
        <Route path="theory-submissions" element={<Navigate to="/admin/theory-exams" replace />} />
      </Route>
    </Routes>
  );
}
