package bootstrap

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cloudctf/backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func OpenDB(databaseURI string) (*gorm.DB, error) {
	dialector, err := dialectorFromURI(databaseURI)
	if err != nil {
		return nil, err
	}
	return gorm.Open(dialector, &gorm.Config{})
}

func dialectorFromURI(databaseURI string) (gorm.Dialector, error) {
	databaseURI = strings.TrimSpace(databaseURI)
	if databaseURI == "" {
		databaseURI = "sqlite://data/cloudctf.db"
	}
	scheme, dsn, ok := strings.Cut(databaseURI, "://")
	if !ok {
		return nil, fmt.Errorf("invalid database uri %q, expected sqlite://, mysql://, postgres://, or postgresql://", databaseURI)
	}
	normalizedScheme := strings.ToLower(scheme)
	switch normalizedScheme {
	case "sqlite", "sqlite3":
		if dsn == "" {
			dsn = "data/cloudctf.db"
		}
		if dsn != ":memory:" {
			if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
				return nil, err
			}
		}
		return sqlite.Open(dsn), nil
	case "mysql":
		if !strings.Contains(dsn, "?") {
			dsn += "?charset=utf8mb4&parseTime=True&loc=Local"
		}
		return mysql.Open(dsn), nil
	case "postgres", "postgresql":
		return postgres.Open(normalizedScheme + "://" + dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database uri scheme %q, expected sqlite://, mysql://, postgres://, or postgresql://", scheme)
	}
}

func PrepareSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.RefreshToken{},
		&models.APIKey{},
		&models.Competition{},
		&models.Team{},
		&models.TeamMember{},
		&models.Challenge{},
		&models.TheoryQuestion{},
		&models.Exam{},
		&models.CompetitionTheoryQuestion{},
		&models.ExamSubmission{},
		&models.Submission{},
		&models.HintPurchase{},
		&models.Scoreboard{},
		&models.Notification{},
		&models.Announcement{},
		&models.AuditLog{},
		&models.VirtualizationHost{},
		&models.VulnLab{},
		&models.ResourceRepo{},
		&models.Writeup{},
		&models.TheorySubmission{},
		&models.CompetitionParticipant{},
		&models.CompetitionAdmin{},
		&models.TeamCompetition{},
		&models.TeamJoinRequest{},
		&models.BuildLog{},
		&models.TemplateSyncLog{},
		&models.SystemSetting{},
		&models.ChallengeVirtualization{},
		&models.ContainerGroup{},
		&models.Templates{},
		&models.Container{},
	); err != nil {
		return err
	}
	return PrepareDatabaseData(db)
}

func PrepareDatabaseData(db *gorm.DB) error {
	if err := seedRoleData(db); err != nil {
		return err
	}
	if err := seedUserData(db); err != nil {
		return err
	}
	if err := seedResourceRepoData(db); err != nil {
		return err
	}
	if err := seedTheoryExamData(db); err != nil {
		return err
	}
	if err := seedNotificationData(db); err != nil {
		return err
	}
	slog.Info("database schema prepared and initial data seeded")
	return nil
}
