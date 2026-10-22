import { useParams } from "react-router-dom";
import { AdminExamDetail } from "../components/AdminExamDetail";

export function AdminExamDetailPage() {
  const { examId = "" } = useParams();
  return (
    <div className="admin-page">
      <AdminExamDetail examId={Number(examId)} backTo="/admin/theory-exams" />
    </div>
  );
}
