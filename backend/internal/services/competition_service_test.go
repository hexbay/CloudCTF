package services

import (
	"testing"

	"cloudctf/backend/internal/bootstrap"
)

func testCompetitionService(t *testing.T) *CompetitionService {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return NewCompetitionService(db)
}
