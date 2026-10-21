import { Spin } from "antd";
import { useEffect, useRef } from "react";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";
import { CompetitionArenaRegistrationGate } from "./CompetitionArenaRegistrationGate";
import { TheoryCompetitionExamPanel } from "./TheoryCompetitionExamPanel";

export function TheoryCompetitionArenaPage() {
  const {
    competition,
    canEnter,
    exam,
    examLoading,
    examForm,
    examAnswers,
    examSubmitting,
    examResult,
    loadExam,
    submitExam,
    navigateToTab,
  } = useCompetitionArena();
  const requestedExamCompetitionRef = useRef<number | null>(null);

  useEffect(() => {
    if (!canEnter || exam || examLoading || !competition?.id) {
      return;
    }
    if (requestedExamCompetitionRef.current === competition.id) {
      return;
    }
    requestedExamCompetitionRef.current = competition.id;
    void loadExam();
  }, [canEnter, competition?.id, exam, examLoading, loadExam]);

  if (!competition) return null;

  if (!canEnter) {
    return <CompetitionArenaRegistrationGate />;
  }

  if (examLoading) {
    return (
      <div style={{ textAlign: "center", padding: "48px 0" }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <TheoryCompetitionExamPanel
      exam={exam}
      endTime={competition.end_time}
      examForm={examForm}
      examAnswers={examAnswers}
      examSubmitting={examSubmitting}
      examResult={examResult}
      onBackToCompetitions={() => navigateToTab("overview")}
      onSubmit={submitExam}
    />
  );
}
