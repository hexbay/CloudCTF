import { useParams } from "react-router-dom";
import { AdminExamDetail } from "../components/AdminExamDetail";

export function AdminCompetitionExamDetailPage() {
  const { competitionId = "", examId = "" } = useParams();
  return (
    <div className="admin-page">
      <AdminExamDetail examId={Number(examId)} competitionId={Number(competitionId)} backTo={`/admin/competition/${competitionId}`} />
    </div>
  );
}
