import { Typography } from "antd";
import { useParams } from "react-router-dom";
import { AdminExamDetail } from "../components/AdminExamDetail";

export function AdminExamDetailPage() {
  const { examId = "" } = useParams();
  return (
    <div className="admin-page">
      <Typography.Title level={4} className="admin-page-title">试卷详情</Typography.Title>
      <AdminExamDetail examId={Number(examId)} backTo="/admin/theory-exams" />
    </div>
  );
}
