package models

import (
	"encoding/json"
	"time"
)

type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	Permissions string    `json:"-"`
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Role) TableName() string { return "role" }

type User struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Email              string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Username           string    `gorm:"size:255;uniqueIndex;not null" json:"username"`
	HashedPassword     string    `gorm:"size:255;not null" json:"-"`
	FullName           string    `gorm:"size:255" json:"full_name"`
	IsActive           bool      `gorm:"not null;default:true" json:"is_active"`
	IsSuperuser        bool      `gorm:"not null;default:false" json:"is_superuser"`
	Tags               string    `json:"-"`
	Avatar             string    `gorm:"size:255" json:"avatar"`
	TeamID             *uint     `gorm:"index" json:"team_id"`
	MustChangePassword bool      `gorm:"not null;default:false" json:"must_change_password"`
	RoleID             *uint     `json:"role_id"`
	Role               Role      `gorm:"foreignKey:RoleID" json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (User) TableName() string { return "user" }

type RefreshToken struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;index" json:"user_id"`
	TokenHash string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

type Competition struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Name             string     `gorm:"size:255;not null" json:"name"`
	Description      string     `json:"description"`
	Type             string     `gorm:"size:32;not null;default:ctf" json:"type"`
	ParticipantMode  string     `gorm:"size:32;not null;default:individual" json:"participant_mode"`
	Status           string     `gorm:"size:32;not null;default:draft" json:"status"`
	StartTime        *time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	IsPublic         bool       `gorm:"not null;default:true" json:"is_public"`
	RegistrationOpen bool       `gorm:"not null;default:false" json:"registration_open"`
	ScoreboardFrozen bool       `gorm:"not null;default:false" json:"scoreboard_frozen"`
	BannerURL        string     `gorm:"size:255" json:"banner_url"`
	Config           string     `json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (Competition) TableName() string { return "competition" }

type Team struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Name            string     `gorm:"size:255;uniqueIndex;not null" json:"name"`
	Description     string     `json:"description"`
	OwnerID         uint       `gorm:"not null;index" json:"owner_id"`
	Owner           User       `gorm:"foreignKey:OwnerID" json:"owner"`
	InvitationCode  string     `gorm:"size:50;uniqueIndex" json:"invitation_code"`
	IsBlacklisted   bool       `gorm:"not null;default:false" json:"is_blacklisted"`
	BlacklistReason string     `gorm:"size:500" json:"blacklist_reason"`
	BlacklistedAt   *time.Time `json:"blacklisted_at"`
	BlacklistedBy   *uint      `json:"blacklisted_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (Team) TableName() string { return "team" }

type TeamMember struct {
	ID       uint      `gorm:"primaryKey"`
	TeamID   uint      `gorm:"not null;index"`
	UserID   uint      `gorm:"not null;index"`
	JoinedAt time.Time `gorm:"not null"`
	LeftAt   *time.Time
	IsActive bool   `gorm:"not null;default:true"`
	Reason   string `gorm:"size:255"`
}

func (TeamMember) TableName() string { return "team_member" }

func (r Role) PermissionList() []string {
	if r.Permissions == "" {
		return []string{}
	}
	items := []string{}
	if err := json.Unmarshal([]byte(r.Permissions), &items); err != nil {
		return []string{}
	}
	return items
}

func (u User) TagList() []string {
	if u.Tags == "" {
		return []string{}
	}
	items := []string{}
	if err := json.Unmarshal([]byte(u.Tags), &items); err != nil {
		return []string{}
	}
	return items
}

// ================================================
// Challenge
// ================================================

type Challenge struct {
	ID                uint         `gorm:"primaryKey" json:"id"`
	Name              string       `gorm:"size:255;not null" json:"name"`
	Description       string       `json:"description"`
	Category          string       `gorm:"size:32;not null" json:"category"`
	Points            int          `gorm:"not null;default:0" json:"points"`
	Difficulty        int          `gorm:"not null;default:1" json:"difficulty"`
	IsActive          bool         `gorm:"not null;default:true" json:"is_active"`
	IsPublic          bool         `gorm:"not null;default:true" json:"is_public"`
	Usage             string       `gorm:"size:32;not null;default:PRACTICE" json:"usage"`
	Flag              string       `gorm:"size:255" json:"-"`
	IsDynamic         bool         `gorm:"not null;default:false" json:"is_dynamic"`
	ContainerImage    string       `gorm:"size:255" json:"container_image"`
	ContainerPort     int          `gorm:"not null;default:0" json:"container_port"`
	ContainerTimeout  int          `gorm:"not null;default:3600" json:"container_timeout"`
	HostID            *uint        `json:"host_id"`
	TemplateID        *uint        `json:"template_id"`
	Template          *Templates   `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	CompetitionID     *uint        `json:"competition_id"`
	Competition       *Competition `gorm:"foreignKey:CompetitionID" json:"competition,omitempty"`
	SourceChallengeID *uint        `json:"source_challenge_id"`
	LevelOneSolution  string       `json:"-"`
	Hints             string       `json:"-"`
	Files             string       `json:"-"`
	CreatedBy         *uint        `json:"created_by"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

func (Challenge) TableName() string { return "challenge" }

func (c Challenge) HintList() []map[string]any {
	return parseJSONArray(c.Hints)
}

func (c Challenge) FileList() []map[string]any {
	return parseJSONArray(c.Files)
}

// ================================================
// Theory Question
// ================================================

type TheoryQuestion struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Question     string    `gorm:"not null" json:"question"`
	Points       int       `gorm:"not null;default:0" json:"points"`
	Category     string    `gorm:"size:100" json:"category"`
	Difficulty   int       `gorm:"not null;default:1" json:"difficulty"`
	QuestionType string    `gorm:"size:32;not null;default:text" json:"question_type"`
	Context      string    `gorm:"not null" json:"-"`
	CreatedBy    *uint     `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (TheoryQuestion) TableName() string { return "theory_question" }

func (t TheoryQuestion) ContextMap() map[string]any {
	return parseJSONObject(t.Context)
}

// ================================================
// Exam
// ================================================

type Exam struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CompetitionID *uint     `gorm:"index" json:"competition_id"`
	Title         string    `gorm:"size:200;not null;index" json:"title"`
	TotalScore    int       `gorm:"not null;default:0" json:"total_score"`
	IsActive      bool      `gorm:"not null;default:true" json:"is_active"`
	Context       string    `gorm:"not null" json:"-"`
	CreatedBy     *uint     `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Exam) TableName() string { return "exam" }

type CompetitionTheoryQuestion struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CompetitionID uint      `gorm:"not null;index:idx_competition_question,unique" json:"competition_id"`
	QuestionID    uint      `gorm:"not null;index:idx_competition_question,unique" json:"question_id"`
	CreatedAt     time.Time `json:"created_at"`
}

func (CompetitionTheoryQuestion) TableName() string {
	return "competition_theory_question"
}

type ExamSubmission struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	CompetitionID uint       `gorm:"not null;index" json:"competition_id"`
	UserID        uint       `gorm:"not null;index" json:"user_id"`
	User          User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TeamID        *uint      `json:"team_id"`
	Team          *Team      `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	ExamID        uint       `gorm:"not null;index" json:"exam_id"`
	Score         int        `gorm:"not null;default:0" json:"score"`
	TotalScore    int        `gorm:"not null;default:0" json:"total_score"`
	Correct       int        `gorm:"not null;default:0" json:"correct"`
	Total         int        `gorm:"not null;default:0" json:"total"`
	Status        string     `gorm:"size:32;not null;default:graded" json:"status"`
	ReviewNote    string     `json:"review_note"`
	Answers       string     `gorm:"not null" json:"-"`
	SubmittedAt   time.Time  `gorm:"not null;index" json:"submitted_at"`
	GradedAt      *time.Time `json:"graded_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (ExamSubmission) TableName() string { return "exam_submission" }

// ================================================
// Submission
// ================================================

type Submission struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ChallengeID   uint      `gorm:"not null;index" json:"challenge_id"`
	UserID        uint      `gorm:"not null;index" json:"user_id"`
	TeamID        *uint     `json:"team_id"`
	CompetitionID *uint     `json:"competition_id"`
	Flag          string    `gorm:"size:255;not null" json:"-"`
	IsCorrect     bool      `gorm:"not null" json:"is_correct"`
	Points        int       `gorm:"not null;default:0" json:"points"`
	IPAddress     string    `gorm:"size:50" json:"ip_address"`
	SubmittedAt   time.Time `gorm:"not null" json:"submitted_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Submission) TableName() string { return "submission" }

type HintPurchase struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index:idx_hint_purchase,unique" json:"user_id"`
	ChallengeID uint      `gorm:"not null;index:idx_hint_purchase,unique" json:"challenge_id"`
	HintIndex   int       `gorm:"not null;index:idx_hint_purchase,unique" json:"hint_index"`
	Cost        int       `gorm:"not null;default:0" json:"cost"`
	CreatedAt   time.Time `json:"created_at"`
}

func (HintPurchase) TableName() string { return "hint_purchase" }

// ================================================
// Scoreboard
// ================================================

type Scoreboard struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	CompetitionID    uint       `gorm:"not null;index" json:"competition_id"`
	TeamID           *uint      `json:"team_id"`
	UserID           *uint      `json:"user_id"`
	TeamScore        float64    `gorm:"not null;default:0" json:"team_score"`
	UserScore        float64    `gorm:"not null;default:0" json:"user_score"`
	SolvedChallenges int        `gorm:"not null;default:0" json:"solved_challenges"`
	LastSubmission   *time.Time `json:"last_submission"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (Scoreboard) TableName() string { return "scoreboard" }

// ================================================
// Notification
// ================================================

type Notification struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"size:255;not null" json:"title"`
	Content     string     `gorm:"not null" json:"content"`
	MessageType string     `gorm:"size:32;not null;default:system" json:"message_type"`
	Audience    string     `gorm:"size:16;not null;default:player;index" json:"audience"`
	Priority    string     `gorm:"size:16;not null;default:normal" json:"priority"`
	SenderID    *uint      `json:"sender_id"`
	SenderName  string     `gorm:"size:100" json:"sender_name"`
	ReceiverID  uint       `gorm:"not null;index" json:"receiver_id"`
	RelatedType string     `gorm:"size:50" json:"related_type"`
	RelatedID   *uint      `json:"related_id"`
	IsRead      bool       `gorm:"not null;default:false" json:"is_read"`
	ReadAt      *time.Time `json:"read_at"`
	SendEmail   bool       `gorm:"not null;default:false" json:"send_email"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (Notification) TableName() string { return "notification" }

// ================================================
// Announcement
// ================================================

type Announcement struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"size:255;not null" json:"title"`
	Content     string     `gorm:"not null" json:"content"`
	Summary     string     `gorm:"size:500" json:"summary"`
	AuthorID    uint       `gorm:"not null" json:"author_id"`
	IsPinned    bool       `gorm:"not null;default:false" json:"is_pinned"`
	PinOrder    int        `gorm:"not null;default:0" json:"pin_order"`
	IsPublished bool       `gorm:"not null;default:false" json:"is_published"`
	TargetType  string     `gorm:"size:50;not null;default:all" json:"target_type"`
	TargetID    *uint      `json:"target_id"`
	Attachments string     `json:"-"`
	ViewCount   int        `gorm:"not null;default:0" json:"view_count"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Announcement) TableName() string { return "announcement" }

// ================================================
// Audit Log
// ================================================

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`                   // 主键
	UserID       *uint     `json:"user_id"`                                // 用户ID（可为空）
	Username     string    `gorm:"size:255" json:"username"`               // 用户名
	ActionType   string    `gorm:"size:50;not null" json:"action_type"`    // 操作类型：login/logout/register/delete/ban/reset_password 等
	ResourceType string    `gorm:"size:100;not null" json:"resource_type"` // 资源类型：user/competition/container 等
	ResourceID   string    `gorm:"size:100" json:"resource_id"`            // 资源ID
	Description  string    `json:"description"`                            // 操作描述
	IPAddress    string    `gorm:"size:50" json:"ip_address"`              // 操作IP
	UserAgent    string    `gorm:"size:255" json:"user_agent"`             // 用户UA
	CreatedAt    time.Time `json:"created_at"`                             // 创建时间
}

func (AuditLog) TableName() string { return "audit_log" }

type TheorySubmission struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	CompetitionID uint       `gorm:"not null;index" json:"competition_id"`
	UserID        uint       `gorm:"not null;index" json:"user_id"`
	TeamID        *uint      `gorm:"index" json:"team_id"`
	ExamID        *uint      `gorm:"index" json:"exam_id"`
	Score         int        `gorm:"not null;default:0" json:"score"`
	Status        string     `gorm:"size:32;not null;default:submitted" json:"status"`
	ReviewNote    string     `json:"review_note"`
	Answers       string     `json:"-"`
	SubmittedAt   *time.Time `json:"submitted_at"`
	GradedAt      *time.Time `json:"graded_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (TheorySubmission) TableName() string { return "theory_submissions" }

type CompetitionParticipant struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CompetitionID uint      `gorm:"not null;index:idx_competition_participant_user,unique" json:"competition_id"`
	UserID        uint      `gorm:"not null;index:idx_competition_participant_user,unique" json:"user_id"`
	Username      string    `gorm:"size:255;not null" json:"username"`
	TeamID        *uint     `gorm:"index" json:"team_id"`
	TeamName      string    `gorm:"size:255" json:"team_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CompetitionParticipant) TableName() string { return "competition_participants" }

type CompetitionAdmin struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CompetitionID uint      `gorm:"not null;index:idx_competition_admin_user,unique" json:"competition_id"`
	UserID        uint      `gorm:"not null;index:idx_competition_admin_user,unique" json:"user_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CompetitionAdmin) TableName() string { return "competition_admins" }

type TeamCompetition struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TeamID        uint      `gorm:"not null;index:idx_team_competition,unique" json:"team_id"`
	CompetitionID uint      `gorm:"not null;index:idx_team_competition,unique" json:"competition_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (TeamCompetition) TableName() string { return "team_competitions" }

type TeamJoinRequest struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TeamID    uint      `gorm:"not null;index" json:"team_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Status    string    `gorm:"size:32;not null;default:approved" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TeamJoinRequest) TableName() string { return "team_join_requests" }

type BuildLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TemplateID   *uint     `gorm:"index" json:"template_id"`
	TemplateName string    `gorm:"size:255" json:"template_name"`
	TaskID       string    `gorm:"size:255;index" json:"task_id"`
	Status       string    `gorm:"size:32;not null" json:"status"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (BuildLog) TableName() string { return "build_logs" }

type TemplateSyncLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TemplateID *uint     `gorm:"index" json:"template_id"`
	Status     string    `gorm:"size:32;not null" json:"status"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (TemplateSyncLog) TableName() string { return "template_sync_logs" }

type SystemSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:128;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"not null" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SystemSetting) TableName() string { return "system_settings" }

type ChallengeVirtualization struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ChallengeID uint      `gorm:"not null;uniqueIndex" json:"challenge_id"`
	Status      string    `gorm:"size:32;not null;default:running" json:"status"`
	HostID      *uint     `gorm:"index" json:"host_id"`
	TemplateID  *uint     `gorm:"index" json:"template_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ChallengeVirtualization) TableName() string { return "challenge_virtualizations" }

// ================================================
// Virtualization Host
// ================================================

type VirtualizationHost struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:100;not null" json:"name"`
	Description string     `json:"description"`
	HostType    string     `gorm:"size:16;not null" json:"host_type"`
	IsDefault   bool       `gorm:"not null;default:false" json:"is_default"`
	Endpoint    string     `gorm:"size:255;not null" json:"endpoint"`
	OutIP       string     `gorm:"size:255" json:"out_ip"`
	APIKey      string     `gorm:"size:255" json:"-"`
	Username    string     `gorm:"size:100" json:"username"`
	Password    string     `gorm:"size:255" json:"-"`
	Status      string     `gorm:"size:16;not null;default:offline" json:"status"`
	LastCheck   *time.Time `json:"last_check"`
	Resources   string     `json:"-"`
	Config      string     `json:"-"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (VirtualizationHost) TableName() string { return "virtualization_host" }

// ================================================
// Vulnerability Lab
// ================================================

type VulnLab struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	CVEID          string    `gorm:"size:64;index" json:"cve_id"`
	Severity       string    `gorm:"size:32" json:"severity"`
	Summary        string    `gorm:"size:512" json:"summary"`
	Description    string    `json:"description"`
	ReproduceSteps string    `json:"reproduce_steps"`
	ReferenceLinks string    `json:"-"`
	IsPublic       bool      `gorm:"not null;default:true" json:"is_public"`
	IsActive       bool      `gorm:"not null;default:true" json:"is_active"`
	TemplateID     uint      `gorm:"not null" json:"template_id"`
	TimeoutSeconds int       `gorm:"not null;default:3600" json:"timeout_seconds"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (VulnLab) TableName() string { return "vuln_lab" }

func (v VulnLab) ReferenceLinkList() []string {
	items := []string{}
	if err := json.Unmarshal([]byte(v.ReferenceLinks), &items); err != nil {
		return []string{}
	}
	return items
}

// ================================================
// Resource Repository
// ================================================

type ResourceRepo struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Name             string     `gorm:"size:255;not null" json:"name"`
	RepoType         string     `gorm:"size:16;not null" json:"repo_type"`
	Provider         string     `gorm:"size:16;not null" json:"provider"`
	RepositoryURL    string     `gorm:"size:1024;not null" json:"repository_url"`
	RepositoryBranch string     `gorm:"size:128;not null;default:main" json:"repository_branch"`
	TemplatesPath    string     `gorm:"size:512" json:"templates_path"`
	HostID           *uint      `json:"host_id"`
	Enabled          bool       `gorm:"not null;default:true" json:"enabled"`
	LastRunAt        *time.Time `json:"last_run_at"`
	Tags             string     `json:"-"`
	Config           string     `json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (ResourceRepo) TableName() string { return "resource_repo" }

func (r ResourceRepo) TagList() []string {
	items := []string{}
	if err := json.Unmarshal([]byte(r.Tags), &items); err != nil {
		return []string{}
	}
	return items
}

// ================================================
// Container Runtime
// ================================================

type ContainerGroup struct {
	ID            uint                 `gorm:"primaryKey" json:"id"`
	Name          string               `gorm:"size:255;not null" json:"name"`
	Description   string               `json:"description"`
	Status        string               `gorm:"size:32;not null;default:pending;index" json:"status"`
	SourceType    string               `gorm:"size:64;index" json:"source_type"`
	HostID        *uint                `gorm:"index" json:"host_id"`
	TemplateID    *uint                `gorm:"index" json:"template_id"`
	TemplateName  string               `gorm:"size:255" json:"template_name"`
	ChallengeID   *uint                `gorm:"index" json:"challenge_id"`
	ChallengeName string               `gorm:"size:255" json:"challenge_name"`
	CompetitionID *uint                `gorm:"index" json:"competition_id"`
	UserID        *uint                `gorm:"index" json:"user_id"`
	UserName      string               `gorm:"size:255" json:"user_name"`
	VulnLabID     *uint                `gorm:"index" json:"vuln_lab_id"`
	VulnLabName   string               `gorm:"size:255" json:"vuln_lab_name"`
	ContainerID   string               `gorm:"size:128;index" json:"container_id"`
	Containers    ContainerRuntimeList `gorm:"type:json" json:"containers"`
	Flag          string               `gorm:"size:255" json:"-"`
	StartedAt     *time.Time           `json:"started_at"`
	ExpiresAt     *time.Time           `gorm:"index" json:"expires_at"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

func (ContainerGroup) TableName() string { return "container_groups" }

type Container struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"size:255;not null" json:"name"`
	Description   string     `json:"description"`
	ContainerID   string     `gorm:"size:128;index" json:"container_id"`
	Status        string     `gorm:"size:32;not null;default:pending;index" json:"status"`
	State         string     `gorm:"size:32" json:"state"`
	GroupID       *uint      `gorm:"index" json:"group_id"`
	HostID        *uint      `gorm:"index" json:"host_id"`
	TemplateID    *uint      `gorm:"index" json:"template_id"`
	ChallengeID   *uint      `gorm:"index" json:"challenge_id"`
	CompetitionID *uint      `gorm:"index" json:"competition_id"`
	UserID        *uint      `gorm:"index" json:"user_id"`
	Image         string     `gorm:"size:255" json:"image"`
	Ports         string     `json:"-"`
	Targets       string     `json:"-"`
	Environment   string     `json:"-"`
	Volumes       string     `json:"-"`
	Resources     string     `json:"-"`
	Flag          string     `gorm:"size:255" json:"-"`
	StartedAt     *time.Time `json:"started_at"`
	ExpiresAt     *time.Time `gorm:"index" json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (Container) TableName() string { return "containers" }

// ================================================
// JSON helpers
// ================================================

func parseJSONArray(raw string) []map[string]any {
	if raw == "" {
		return []map[string]any{}
	}
	items := []map[string]any{}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []map[string]any{}
	}
	return items
}

func parseJSONObject(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	obj := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return map[string]any{}
	}
	return obj
}

type TemplateService struct {
	Image      string   `json:"image"`      // 服务镜像
	Name       string   `json:"name"`       // 服务名称
	Ports      []string `json:"ports"`      // 服务暴露端口
	Dockerfile string   `json:"dockerfile"` // Dockerfile 内容
}

type TemplateStatus string

const (
	TemplateStatusPending   TemplateStatus = "pending"
	TemplateStatusReady     TemplateStatus = "ready"
	TemplateStatusBuilding  TemplateStatus = "building"
	TemplateStatusCompleted TemplateStatus = "completed"
)

// Templates 模板/漏洞环境定义，从资源仓库同步
type Templates struct {
	Id            uint                `gorm:"primaryKey" json:"id"` // 主键
	CreatedAt     time.Time           `json:"created_at"`           // 创建时间
	UpdatedAt     time.Time           `json:"updated_at"`           // 更新时间
	ComposeFile   string              `json:"compose_file"`         // docker-compose 文件内容
	Path          string              `gorm:"size:512" json:"path"` // compose 文件在仓库中的相对路径
	Description   string              `json:"description"`          // 描述说明
	HostId        *uint               `gorm:"index" json:"host_id"` // 关联的虚拟化主机 ID
	Host          *VirtualizationHost `gorm:"foreignKey:HostId" json:"host,omitempty"`
	LastSyncAt    time.Time           `json:"last_sync_at"`                           // 最后一次从仓库同步的时间
	Name          string              `gorm:"size:255;not null" json:"name"`          // 模板名称
	Ports         []string            `gorm:"type:json;serializer:json" json:"ports"` // 容器端口列表
	RepositoryUrl string              `gorm:"size:1024" json:"repository_url"`        // 来源仓库地址 URL
	Services      []TemplateService   `gorm:"type:json;serializer:json" json:"services"`
	Status        TemplateStatus      `gorm:"size:32;not null;default:pending;index" json:"status"`  // 状态：pending / building / ready / error
	Tags          []string            `gorm:"type:json;serializer:json" json:"tags"`                 // 标签列表
	TemplateType  string              `gorm:"size:32;not null;default:compose" json:"template_type"` // 模板类型：compose / image / k8s 等
}

// TableName 返回模板表名
func (Templates) TableName() string { return "templates" }
