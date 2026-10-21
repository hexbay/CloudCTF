package services

import (
	"fmt"
	"math/rand"
	"time"

	"cloudctf/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type DemoDataSummary struct {
	Scale        string   `json:"scale"`
	Users        int      `json:"users"`
	Teams        int      `json:"teams"`
	Competitions int      `json:"competitions"`
	Challenges   int      `json:"challenges"`
	Submissions  int      `json:"submissions"`
	Types        []string `json:"types"`
}

type DemoDataService struct {
	db *gorm.DB
}

func NewDemoDataService(db *gorm.DB) *DemoDataService {
	return &DemoDataService{db: db}
}

type demoScaleConfig struct {
	Key                    string
	UserCount              int
	TeamCount              int
	ParticipantsPerContest int
	SubmissionsPerUserMin  int
	SubmissionsPerUserMax  int
}

type demoCompetitionSpec struct {
	Key              string
	Name             string
	Description      string
	Type             string
	ParticipantMode  string
	Status           string
	StartOffset      time.Duration
	EndOffset        time.Duration
	IsPublic         bool
	RegistrationOpen bool
}

func (s *DemoDataService) Generate(scale string) (DemoDataSummary, error) {
	cfg, err := demoScaleByKey(scale)
	if err != nil {
		return DemoDataSummary{}, err
	}

	summary := DemoDataSummary{Scale: cfg.Key}
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))

	err = s.db.Transaction(func(tx *gorm.DB) error {
		roleID, err := ensureDemoPlayerRole(tx)
		if err != nil {
			return err
		}

		users, addedUsers, err := ensureDemoUsers(tx, roleID, cfg)
		if err != nil {
			return err
		}
		summary.Users = addedUsers

		teams, teamOrder, addedTeams, err := ensureDemoTeams(tx, users, cfg)
		if err != nil {
			return err
		}
		summary.Teams = addedTeams

		comps, addedCompetitions, err := ensureDemoCompetitions(tx, cfg)
		if err != nil {
			return err
		}
		summary.Competitions = addedCompetitions

		challenges, addedChallenges, types, err := ensureDemoChallenges(tx, comps)
		if err != nil {
			return err
		}
		summary.Challenges = addedChallenges
		summary.Types = types

		addedSubmissions, err := ensureDemoSubmissions(tx, seed, users, teamOrder, teams, comps, challenges, cfg)
		if err != nil {
			return err
		}
		summary.Submissions = addedSubmissions
		return nil
	})

	return summary, err
}

func demoScaleByKey(scale string) (demoScaleConfig, error) {
	switch scale {
	case "2-10":
		return demoScaleConfig{Key: "2-10", UserCount: 12, TeamCount: 4, ParticipantsPerContest: 6, SubmissionsPerUserMin: 1, SubmissionsPerUserMax: 3}, nil
	case "10-100":
		return demoScaleConfig{Key: "10-100", UserCount: 48, TeamCount: 16, ParticipantsPerContest: 24, SubmissionsPerUserMin: 2, SubmissionsPerUserMax: 5}, nil
	case "100+":
		return demoScaleConfig{Key: "100+", UserCount: 180, TeamCount: 60, ParticipantsPerContest: 100, SubmissionsPerUserMin: 3, SubmissionsPerUserMax: 7}, nil
	default:
		return demoScaleConfig{}, fmt.Errorf("unsupported scale: %s", scale)
	}
}

func ensureDemoPlayerRole(tx *gorm.DB) (*uint, error) {
	var role models.Role
	if err := tx.Where("name = ?", "player").First(&role).Error; err == nil {
		return &role.ID, nil
	}
	role = models.Role{Name: "player", Description: "Player", Permissions: "[]", IsSystem: true}
	if err := tx.Create(&role).Error; err != nil {
		return nil, err
	}
	return &role.ID, nil
}

func ensureDemoUsers(tx *gorm.DB, roleID *uint, cfg demoScaleConfig) (map[string]models.User, int, error) {
	firstNames := []string{"Alex", "Blake", "Casey", "Drew", "Evan", "Flynn", "Gray", "Harper", "Indy", "Jules", "Kai", "Logan"}
	lastNames := []string{"Wang", "Li", "Chen", "Liu", "Zhang", "Zhao", "Sun", "Xu", "Lin", "Guo", "He", "Ma"}
	hashed, err := bcrypt.GenerateFromPassword([]byte("demo123456"), bcrypt.DefaultCost)
	if err != nil {
		return nil, 0, err
	}

	users := make(map[string]models.User, cfg.UserCount)
	added := 0
	for i := 1; i <= cfg.UserCount; i++ {
		username := fmt.Sprintf("demo_%s_user_%03d", scaleToken(cfg.Key), i)
		var user models.User
		err := tx.Preload("Role").Where("username = ?", username).First(&user).Error
		if err == nil {
			users[username] = user
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, 0, err
		}
		user = models.User{
			Username:       username,
			Email:          fmt.Sprintf("%s@demo.local", username),
			FullName:       fmt.Sprintf("%s %s", firstNames[(i-1)%len(firstNames)], lastNames[(i-1)%len(lastNames)]),
			HashedPassword: string(hashed),
			IsActive:       true,
			IsSuperuser:    false,
			Tags:           `["demo","seed"]`,
			RoleID:         roleID,
		}
		if err := tx.Create(&user).Error; err != nil {
			return nil, 0, err
		}
		users[username] = user
		added++
	}
	return users, added, nil
}

func ensureDemoTeams(tx *gorm.DB, users map[string]models.User, cfg demoScaleConfig) (map[string]models.Team, []string, int, error) {
	prefixes := []string{"Red", "Blue", "Orange", "Purple", "Silver", "Gold", "Byte", "Root", "Kernel", "Cipher", "Shadow", "Vector"}
	suffixes := []string{"Team", "Squad", "Crew", "Alliance", "Lab", "Unit"}
	teams := make(map[string]models.Team, cfg.TeamCount)
	order := make([]string, 0, cfg.TeamCount)
	added := 0
	membersPerTeam := maxInt(2, cfg.UserCount/cfg.TeamCount)

	for i := 0; i < cfg.TeamCount; i++ {
		teamName := fmt.Sprintf("Demo %s %s %02d", prefixes[i%len(prefixes)], suffixes[i%len(suffixes)], i+1)
		order = append(order, teamName)
		var team models.Team
		err := tx.Preload("Owner").Where("name = ?", teamName).First(&team).Error
		if err == nil {
			teams[teamName] = team
		} else if err != nil && err != gorm.ErrRecordNotFound {
			return nil, nil, 0, err
		} else {
			owner := users[fmt.Sprintf("demo_%s_user_%03d", scaleToken(cfg.Key), i*membersPerTeam+1)]
			team = models.Team{Name: teamName, Description: fmt.Sprintf("%s seeded for scale %s", teamName, cfg.Key), OwnerID: owner.ID, InvitationCode: fmt.Sprintf("DEMO-%s-%02d", scaleToken(cfg.Key), i+1)}
			if err := tx.Create(&team).Error; err != nil {
				return nil, nil, 0, err
			}
			added++
			teams[teamName] = team
		}

		for memberOffset := 0; memberOffset < membersPerTeam; memberOffset++ {
			userIndex := i*membersPerTeam + memberOffset + 1
			if userIndex > cfg.UserCount {
				break
			}
			username := fmt.Sprintf("demo_%s_user_%03d", scaleToken(cfg.Key), userIndex)
			member := users[username]
			var existing models.TeamMember
			err := tx.Where("team_id = ? AND user_id = ? AND is_active = ?", team.ID, member.ID, true).First(&existing).Error
			if err == nil {
				continue
			}
			if err != nil && err != gorm.ErrRecordNotFound {
				return nil, nil, 0, err
			}
			if err := tx.Create(&models.TeamMember{TeamID: team.ID, UserID: member.ID, JoinedAt: time.Now().Add(-time.Duration(userIndex) * time.Hour), IsActive: true, Reason: "demo seed data"}).Error; err != nil {
				return nil, nil, 0, err
			}
		}
	}
	return teams, order, added, nil
}

func ensureDemoCompetitions(tx *gorm.DB, cfg demoScaleConfig) ([]models.Competition, int, error) {
	now := time.Now()
	specs := []demoCompetitionSpec{
		{Key: "beginner-individual", Name: fmt.Sprintf("Demo Beginner Individual %s", cfg.Key), Description: "新手个人赛演示数据", Type: "ctf", ParticipantMode: "individual", Status: "finished", StartOffset: -10 * 24 * time.Hour, EndOffset: -9 * 24 * time.Hour, IsPublic: true, RegistrationOpen: false},
		{Key: "team-jeopardy", Name: fmt.Sprintf("Demo Team Jeopardy %s", cfg.Key), Description: "团队解题赛演示数据", Type: "mixed", ParticipantMode: "team", Status: "ongoing", StartOffset: -6 * time.Hour, EndOffset: 18 * time.Hour, IsPublic: true, RegistrationOpen: true},
		{Key: "attack-defense", Name: fmt.Sprintf("Demo Attack Defense %s", cfg.Key), Description: "攻防赛演示数据", Type: "awd", ParticipantMode: "team", Status: "upcoming", StartOffset: 48 * time.Hour, EndOffset: 72 * time.Hour, IsPublic: true, RegistrationOpen: true},
		{Key: "theory-quiz", Name: fmt.Sprintf("Demo Theory Quiz %s", cfg.Key), Description: "理论赛演示数据", Type: "theory", ParticipantMode: "individual", Status: "draft", StartOffset: 96 * time.Hour, EndOffset: 120 * time.Hour, IsPublic: false, RegistrationOpen: false},
	}

	comps := make([]models.Competition, 0, len(specs))
	added := 0
	for _, spec := range specs {
		var comp models.Competition
		err := tx.Where("name = ?", spec.Name).First(&comp).Error
		if err == nil {
			comps = append(comps, comp)
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, 0, err
		}
		start := now.Add(spec.StartOffset)
		end := now.Add(spec.EndOffset)
		comp = models.Competition{Name: spec.Name, Description: spec.Description, Type: spec.Type, ParticipantMode: spec.ParticipantMode, Status: spec.Status, StartTime: &start, EndTime: &end, IsPublic: spec.IsPublic, RegistrationOpen: spec.RegistrationOpen, Config: fmt.Sprintf(`{"demo_scale":"%s","demo_type":"%s"}`, cfg.Key, spec.Key)}
		if err := tx.Create(&comp).Error; err != nil {
			return nil, 0, err
		}
		comps = append(comps, comp)
		added++
	}
	return comps, added, nil
}

func ensureDemoChallenges(tx *gorm.DB, comps []models.Competition) ([]models.Challenge, int, []string, error) {
	base := []struct {
		Name        string
		Category    string
		Points      int
		Difficulty  int
		Description string
		Flag        string
	}{
		{"Welcome Checkin", "misc", 50, 1, "A warm-up challenge for demo data", "flag{demo_welcome}"},
		{"SQL Injection 101", "web", 100, 2, "Classic login bypass", "flag{demo_sql}"},
		{"Packet Trace", "misc", 120, 2, "Inspect the packet capture", "flag{demo_pcap}"},
		{"Easy Crypto", "crypto", 150, 2, "Basic Caesar and base64", "flag{demo_crypto}"},
		{"Reverse Me", "reverse", 200, 3, "Lightweight reverse engineering", "flag{demo_reverse}"},
		{"Stack Smash", "pwn", 250, 4, "Simple buffer overflow", "flag{demo_pwn}"},
		{"Blue Team Triage", "misc", 180, 3, "Analyze alerts and extract impact", "flag{demo_blue}"},
		{"Service Hardening", "web", 220, 3, "Patch the exposed service", "flag{demo_harden}"},
	}

	challenges := make([]models.Challenge, 0, len(comps)*len(base))
	added := 0
	typeSet := map[string]struct{}{}
	for _, comp := range comps {
		typeSet[comp.Type] = struct{}{}
		for _, item := range base {
			name := fmt.Sprintf("%s - %s", comp.Name, item.Name)
			var challenge models.Challenge
			err := tx.Where("competition_id = ? AND name = ?", comp.ID, name).First(&challenge).Error
			if err == nil {
				challenges = append(challenges, challenge)
				continue
			}
			if err != nil && err != gorm.ErrRecordNotFound {
				return nil, 0, nil, err
			}
			challenge = models.Challenge{Name: name, Description: item.Description, Category: item.Category, Points: item.Points, Difficulty: item.Difficulty, IsActive: true, IsPublic: comp.Status != "draft", Usage: "competition", Flag: item.Flag, IsDynamic: false, CompetitionID: &comp.ID}
			if err := tx.Create(&challenge).Error; err != nil {
				return nil, 0, nil, err
			}
			challenges = append(challenges, challenge)
			added++
		}
	}
	types := make([]string, 0, len(typeSet))
	for typ := range typeSet {
		types = append(types, typ)
	}
	return challenges, added, types, nil
}

func ensureDemoSubmissions(tx *gorm.DB, seed *rand.Rand, users map[string]models.User, teamOrder []string, teams map[string]models.Team, comps []models.Competition, challenges []models.Challenge, cfg demoScaleConfig) (int, error) {
	challengeByCompetition := make(map[uint][]models.Challenge)
	for _, challenge := range challenges {
		if challenge.CompetitionID != nil {
			challengeByCompetition[*challenge.CompetitionID] = append(challengeByCompetition[*challenge.CompetitionID], challenge)
		}
	}
	userList := make([]models.User, 0, len(users))
	for i := 1; i <= cfg.UserCount; i++ {
		userList = append(userList, users[fmt.Sprintf("demo_%s_user_%03d", scaleToken(cfg.Key), i)])
	}
	added := 0
	teamCursor := 0
	for _, comp := range comps {
		if comp.Status == "draft" || comp.Status == "upcoming" {
			continue
		}
		compChallenges := challengeByCompetition[comp.ID]
		if len(compChallenges) == 0 || comp.StartTime == nil || comp.EndTime == nil {
			continue
		}
		participantCount := minInt(cfg.ParticipantsPerContest, len(userList))
		for index := 0; index < participantCount; index++ {
			user := userList[index]
			solveLimit := cfg.SubmissionsPerUserMin + seed.Intn(cfg.SubmissionsPerUserMax-cfg.SubmissionsPerUserMin+1)
			if solveLimit > len(compChallenges) {
				solveLimit = len(compChallenges)
			}
			for solveIndex := 0; solveIndex < solveLimit; solveIndex++ {
				challenge := compChallenges[(index+solveIndex)%len(compChallenges)]
				var existing models.Submission
				err := tx.Where("challenge_id = ? AND user_id = ? AND competition_id = ? AND is_correct = ?", challenge.ID, user.ID, comp.ID, true).First(&existing).Error
				if err == nil {
					continue
				}
				if err != nil && err != gorm.ErrRecordNotFound {
					return 0, err
				}
				submittedAt := comp.StartTime.Add(time.Duration(index*13+solveIndex*17) * time.Minute)
				if submittedAt.After(*comp.EndTime) {
					submittedAt = comp.EndTime.Add(-time.Duration((solveIndex%10)+1) * time.Minute)
				}
				var teamID *uint
				if comp.ParticipantMode == "team" && len(teamOrder) > 0 {
					teamName := teamOrder[teamCursor%len(teamOrder)]
					team := teams[teamName]
					teamID = &team.ID
				}
				if err := tx.Create(&models.Submission{ChallengeID: challenge.ID, UserID: user.ID, TeamID: teamID, CompetitionID: &comp.ID, Flag: challenge.Flag, IsCorrect: true, Points: challenge.Points, IPAddress: fmt.Sprintf("10.%d.%d.%d", (index%200)+1, (solveIndex%200)+1, (teamCursor%200)+1), SubmittedAt: submittedAt}).Error; err != nil {
					return 0, err
				}
				added++
			}
			if comp.ParticipantMode == "team" {
				teamCursor++
			}
		}
	}
	return added, nil
}

func scaleToken(scale string) string {
	switch scale {
	case "2-10":
		return "s"
	case "10-100":
		return "m"
	case "100+":
		return "l"
	default:
		return "x"
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
