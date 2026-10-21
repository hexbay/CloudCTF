package httpserver

import (
	"errors"
	"fmt"

	"cloudctf/backend/internal/services"
)

func (srv *Server) checkCompetitionChallengeStartAllowed(ch challengeModel, user userModel) error {
	if ch.CompetitionID == nil {
		return nil
	}
	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		return fmt.Errorf("service unavailable")
	}
	return relationService.EnsureCanAccessCompetition(user.ID, *ch.CompetitionID)
}

func (srv *Server) ensureCompetitionChallengeStartAllowed(ch challengeModel, user userModel) error {
	return srv.checkCompetitionChallengeStartAllowed(ch, user)
}

func (srv *Server) checkCompetitionExamAllowed(competitionID uint, user userModel) error {
	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		return fmt.Errorf("service unavailable")
	}
	return relationService.EnsureCanAccessExam(user.ID, competitionID)
}

func (srv *Server) checkCompetitionExamSubmitAllowed(competitionID uint, user userModel) error {
	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		return fmt.Errorf("service unavailable")
	}
	return relationService.EnsureCanSubmitExam(user.ID, competitionID)
}

var _ = errors.Is
var _ = services.ErrForbidden
