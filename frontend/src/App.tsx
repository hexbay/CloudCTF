import { Spin } from 'antd';
import { lazy, Suspense } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { NotFoundPage } from './shared/pages/NotFoundPage';

const AdminApp = lazy(() => import('./apps/admin/AdminApp').then((module) => ({ default: module.AdminApp })));
const PlayerApp = lazy(() => import('./apps/player/PlayerApp').then((module) => ({ default: module.PlayerApp })));

export default function App() {
  return (
    <Suspense fallback={<AppLoading />}>
      <Routes>
        <Route path="/404" element={<NotFoundPage />} />
        <Route path="/admin/*" element={<AdminApp />} />
        <Route path="/*" element={<PlayerApp />} />
        <Route path="*" element={<Navigate to="/404" replace />} />
      </Routes>
    </Suspense>
  );
}

function AppLoading() {
  return (
    <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center' }}>
      <Spin size="large" tip="加载中" />
    </div>
  );
}
