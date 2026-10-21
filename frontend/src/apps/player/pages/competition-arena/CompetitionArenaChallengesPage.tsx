import { Spin } from "antd";
import { useEffect } from "react";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";
import { AwdCompetitionArenaPage } from "./AwdCompetitionArenaPage";
import { CompetitionArenaRegistrationGate } from "./CompetitionArenaRegistrationGate";
import { CtfCompetitionArenaPage } from "./CtfCompetitionArenaPage";
import { TheoryCompetitionArenaPage } from "./TheoryCompetitionArenaPage";

export function CompetitionArenaChallengesPage() {
  const {
    competition,
    canEnter,
    challengeMode,
    challenges,
    challengesLoading,
    loadChallenges,
  } = useCompetitionArena();

  useEffect(() => {
    if (canEnter && challengeMode !== "theory" && challenges.length === 0 && !challengesLoading) {
      void loadChallenges();
    }
  }, [canEnter, challengeMode, challenges.length, challengesLoading, loadChallenges]);

  if (!competition) return null;

  if (challengeMode === "theory") {
    return <TheoryCompetitionArenaPage />;
  }

  if (!canEnter) {
    return <CompetitionArenaRegistrationGate />;
  }

  if (challengesLoading) {
    return (
      <div style={{ textAlign: "center", padding: "48px 0" }}>
        <Spin size="large" />
      </div>
    );
  }

  if (challengeMode === "awd") {
    return <AwdCompetitionArenaPage />;
  }

  return <CtfCompetitionArenaPage />;
}
