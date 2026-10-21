import React from 'react';
import ReactDOM from 'react-dom/client';
import { App as AntdApp, ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { BrowserRouter } from 'react-router-dom';
import RootApp from './App';
import { API_ERROR_EVENT } from './shared/api/http';
import './styles/global.css';
import './styles/admin.css';
import './styles/player.css';
import './styles/player-competition.css';
import './styles/player-scoreboard.css';
import './styles/player-vulnlabs.css';

function ApiErrorNotifier() {
  const { message } = AntdApp.useApp();

  React.useEffect(() => {
    function handleApiError(event: Event) {
      const detail = (event as CustomEvent<{ message?: string }>).detail;
      message.error(detail?.message || '请求失败');
    }

    window.addEventListener(API_ERROR_EVENT, handleApiError);
    return () => window.removeEventListener(API_ERROR_EVENT, handleApiError);
  }, [message]);

  return null;
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#0f766e',
          borderRadius: 6,
        },
        components: {
          Menu: {
            darkItemBg: '#0b1624',
            darkSubMenuItemBg: '#0b1624',
            darkItemSelectedBg: '#0e7490',
            darkItemHoverBg: 'rgba(255, 255, 255, 0.08)',
          },
          Layout: {
            siderBg: '#0b1624',
            headerBg: '#0b1624',
          },
        },
      }}
    >
      <AntdApp>
        <ApiErrorNotifier />
        <BrowserRouter>
          <RootApp />
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  </React.StrictMode>
);
