import type { ReactNode } from 'react';
import { Typography } from 'antd';

type PlayerPageProps = {
  icon?: ReactNode;
  title: string;
  subtitle?: string;
  extra?: ReactNode;
  showTitle?: boolean;
  children: ReactNode;
};

export function PlayerPage({ icon, title, subtitle, extra, showTitle = false, children }: PlayerPageProps) {
  return (
    <section className="player-page">
      <div className="player-page-header">
        <div>
          {showTitle ? (
            <Typography.Title level={2} className="player-page-title">
              {icon}
              {title}
            </Typography.Title>
          ) : null}
          {subtitle ? <Typography.Paragraph className="player-page-subtitle">{subtitle}</Typography.Paragraph> : null}
        </div>
        {extra ? <div className="player-page-extra">{extra}</div> : null}
      </div>
      {children}
    </section>
  );
}
