import { Typography } from "antd";
import { useParams } from "react-router-dom";
import { AdminExamDetail } from "../components/AdminExamDetail";

export function AdminCompetitionExamDetailPage() {
  const { competitionId = "", examId = "" } = useParams();
  return (
    <div className="admin-page">
      <Typography.Title level={4} className="admin-page-title">赛事试卷详情</Typography.Title>
      <AdminExamDetail examId={Number(examId)} competitionId={Number(competitionId)} backTo={`/admin/competition/${competitionId}`} />
    </div>
  );
}
