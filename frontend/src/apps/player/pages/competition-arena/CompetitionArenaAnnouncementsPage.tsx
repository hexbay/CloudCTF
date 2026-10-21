import { Empty, List, Card } from "antd";
import { NotificationOutlined } from "@ant-design/icons";
import { formatDate } from "./TheoryCompetitionExamPanel";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";

export function CompetitionArenaAnnouncementsPage() {
  const { announcements } = useCompetitionArena();

  return (
    <Card className="player-panel-card arena-announcement-card" title="比赛公告">
      {announcements.length ? (
        <List
          className="arena-announcement-list"
          dataSource={announcements}
          renderItem={(announcement) => (
            <List.Item>
              <List.Item.Meta
                avatar={<NotificationOutlined />}
                title={announcement.title}
                description={
                  <>
                    <div>{announcement.summary || announcement.content || "暂无公告内容"}</div>
                    <span>{formatDate(announcement.published_at || announcement.created_at)}</span>
                  </>
                }
              />
            </List.Item>
          )}
        />
      ) : (
        <Empty description="暂无比赛公告" />
      )}
    </Card>
  );
}
