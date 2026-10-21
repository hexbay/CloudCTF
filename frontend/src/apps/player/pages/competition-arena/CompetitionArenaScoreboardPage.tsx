import { useEffect, useRef } from "react";
import { CtfScoreboardPanel } from "./CtfScoreboardPanel";
import { TheoryExamScoreboardPanel } from "./TheoryExamScoreboardPanel";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";

export function CompetitionArenaScoreboardPage() {
  const { ranks, competition, scoreboardLoading, isTheoryCompetition, loadScoreboard } = useCompetitionArena();
  const requestedScoreboardCompetitionRef = useRef<number | null>(null);

  useEffect(() => {
    if (!competition?.id || ranks.length > 0 || scoreboardLoading) {
      return;
    }
    if (requestedScoreboardCompetitionRef.current === competition.id) {
      return;
    }
    requestedScoreboardCompetitionRef.current = competition.id;
    void loadScoreboard();
  }, [competition?.id, ranks.length, scoreboardLoading, loadScoreboard]);

  if (isTheoryCompetition) {
    return (
      <TheoryExamScoreboardPanel
        ranks={ranks}
        loading={scoreboardLoading}
        competitionName={competition?.name}
      />
    );
  }

  return (
    <CtfScoreboardPanel
      ranks={ranks}
      loading={scoreboardLoading}
      competitionName={competition?.name}
    />
  );
}
