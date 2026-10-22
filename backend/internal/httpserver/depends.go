package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"cloudctf/backend/internal/config"
	"cloudctf/backend/internal/services"
	"cloudctf/backend/internal/updater"

	"gorm.io/gorm"
)

// Dependencies is the set of external dependencies required to build a Server.
type Dependencies struct {
	Config         config.Config
	Logger         *slog.Logger
	DB             *gorm.DB
	Version        string
	RequestRestart func()
}

// Server is the composition root for the HTTP layer. It owns configuration,
// the database handle, the logger and all initialized services.
type Server struct {
	config          config.Config
	logger          *slog.Logger
	db              *gorm.DB
	version         string
	updater         *updater.Service
	requestRestart  func()
	services        serviceRegistry
	templates       *templateResourceStore
	bootedAt        time.Time
	streamTicketsMu sync.Mutex
	streamTickets   map[string]streamTicket
	updateJob       updateJobState
}

// NewServer builds a fully initialized Server from its dependencies.
func NewServer(deps Dependencies) *Server {
	srv := &Server{
		config:  deps.Config,
		logger:  deps.Logger,
		db:      deps.DB,
		version: deps.Version,
		updater: updater.New(updater.Config{
			Repository:    deps.Config.UpdateRepository,
			APIBaseURL:    deps.Config.UpdateAPIURL,
			Token:         deps.Config.UpdateGitHubToken,
			Proxy:         deps.Config.Proxy,
			Enabled:       deps.Config.UpdateEnabled,
			Version:       deps.Version,
			Persistent:    updater.RunningInContainer(),
			PersistentDir: deps.Config.UpdateDir,
		}),
		requestRestart: deps.RequestRestart,
		bootedAt:       time.Now(),
		streamTickets:  make(map[string]streamTicket),
	}
	srv.initServices(deps.DB)
	return srv
}

type serviceRegistry struct {
	announcementService            *services.AnnouncementService
	authService                    *services.AuthService
	auditLogService                *services.AuditLogService
	backupService                  *services.BackupService
	challengeService               *services.ChallengeService
	challengeVirtualizationService *services.ChallengeVirtualizationService
	competitionRelationService     *services.CompetitionRelationService
	competitionService             *services.CompetitionService
	containerGroupService          *services.ContainerGroupService
	demoDataService                *services.DemoDataService
	examService                    *services.ExamService
	flagService                    *services.FlagService
	miscResourceService            *services.MiscResourceService
	notificationService            *services.NotificationService
	resourceRepositoryService      *services.ResourceRepositoryService
	roleService                    *services.RoleService
	scoreboardService              *services.ScoreboardService
	submissionService              *services.SubmissionService
	systemSettingService           *services.SystemSettingService
	teamService                    *services.TeamService
	templateRuntimeService         *services.TemplateRuntimeService
	theoryService                  *services.TheoryService
	theorySubmissionService        *services.TheorySubmissionService
	userService                    *services.UserService
	virtualizationHostService      *services.VirtualizationHostService
	vulnLabService                 *services.VulnLabService
	writeupService                 *services.WriteupService
}

func (srv *Server) currentConfig() config.Config {
	return srv.config
}

func (srv *Server) currentVersion() string {
	if srv.version == "" {
		return "dev"
	}
	return srv.version
}

func (srv *Server) currentServices() *serviceRegistry {
	return &srv.services
}

func (srv *Server) currentBootedAt() time.Time {
	return srv.bootedAt
}

// initServices wires every service into the registry in one place.
// Order matters only where a service depends on another (e.g. authService
// needs userService), so dependents are created after their dependencies.
func (s *Server) initServices(db *gorm.DB) {
	reg := &s.services

	reg.containerGroupService = services.NewContainerGroupService(db, containerGroupRuntimeAdapter{srv: s})
	reg.virtualizationHostService = services.NewVirtualizationHostService(db)
	reg.resourceRepositoryService = services.NewResourceRepositoryService(db)
	reg.vulnLabService = services.NewVulnLabService(db)
	reg.templateRuntimeService = services.NewTemplateRuntimeService(db, s.config.Proxy)
	s.templates = &templateResourceStore{db: db, srv: s}
	reg.writeupService = services.NewWriteupService(db)
	reg.challengeVirtualizationService = services.NewChallengeVirtualizationService(db)
	reg.flagService = services.NewFlagService(db)
	reg.challengeService = services.NewChallengeService(db)
	reg.announcementService = services.NewAnnouncementService(db)
	reg.notificationService = services.NewNotificationService(db)
	reg.competitionService = services.NewCompetitionService(db)
	reg.competitionRelationService = services.NewCompetitionRelationService(db)
	reg.miscResourceService = services.NewMiscResourceService(db)
	reg.systemSettingService = services.NewSystemSettingService(db)
	reg.teamService = services.NewTeamService(db)
	reg.submissionService = services.NewSubmissionService(db)
	reg.theorySubmissionService = services.NewTheorySubmissionService(db)
	reg.auditLogService = services.NewAuditLogService(db)
	reg.scoreboardService = services.NewScoreboardService(db)
	reg.userService = services.NewUserService(db)
	reg.roleService = services.NewRoleService(db)
	reg.authService = services.NewAuthService(reg.userService, httpTokenIssuer{srv: s})
	reg.theoryService = services.NewTheoryService(db)
	reg.examService = services.NewExamService(db)
	reg.demoDataService = services.NewDemoDataService(db)
	reg.backupService = services.NewBackupService(db)
}

func (srv *Server) getAnnouncementService() *services.AnnouncementService {
	return srv.currentServices().announcementService
}

func (srv *Server) getAuthService() *services.AuthService {
	return srv.currentServices().authService
}

func (srv *Server) getAuditLogService() *services.AuditLogService {
	return srv.currentServices().auditLogService
}

func (srv *Server) auditLoginSuccess(userID uint, username, role string, r *http.Request) {
	description := fmt.Sprintf("%s登录成功", roleText(role))
	resourceType := fmt.Sprintf("%s_user", role)
	_, _ = srv.getAuditLogService().Create(map[string]any{
		"user_id":       userID,
		"username":      username,
		"action_type":   "login",
		"resource_type": resourceType,
		"resource_id":   fmt.Sprintf("%d", userID),
		"description":   description,
		"ip_address":    r.RemoteAddr,
		"user_agent":    r.UserAgent(),
	})
}

func (srv *Server) auditLoginFailed(username, role string, r *http.Request) {
	description := fmt.Sprintf("%s登录失败: %s", roleText(role), username)
	resourceType := fmt.Sprintf("%s_user", role)
	_, _ = srv.getAuditLogService().Create(map[string]any{
		"username":      username,
		"action_type":   "login_failed",
		"resource_type": resourceType,
		"resource_id":   username,
		"description":   description,
		"ip_address":    r.RemoteAddr,
		"user_agent":    r.UserAgent(),
	})
}

func roleText(role string) string {
	if role == "admin" {
		return "管理员"
	}
	return "用户"
}

func (srv *Server) getBackupService() *services.BackupService {
	return srv.currentServices().backupService
}

func (srv *Server) getChallengeService() *services.ChallengeService {
	return srv.currentServices().challengeService
}

func (srv *Server) getChallengeVirtualizationService() *services.ChallengeVirtualizationService {
	return srv.currentServices().challengeVirtualizationService
}

func (srv *Server) getCompetitionRelationService() *services.CompetitionRelationService {
	return srv.currentServices().competitionRelationService
}

func (srv *Server) getCompetitionService() *services.CompetitionService {
	return srv.currentServices().competitionService
}

func (srv *Server) getDemoDataService() *services.DemoDataService {
	return srv.currentServices().demoDataService
}

func (srv *Server) getExamService() *services.ExamService {
	return srv.currentServices().examService
}

func (srv *Server) getFlagService() *services.FlagService {
	return srv.currentServices().flagService
}

func (srv *Server) getMiscResourceService() *services.MiscResourceService {
	return srv.currentServices().miscResourceService
}

func (srv *Server) getNotificationService() *services.NotificationService {
	return srv.currentServices().notificationService
}

func (srv *Server) getResourceRepositoryService() *services.ResourceRepositoryService {
	return srv.currentServices().resourceRepositoryService
}

func (srv *Server) getRoleService() *services.RoleService {
	return srv.currentServices().roleService
}

func (srv *Server) getScoreboardService() *services.ScoreboardService {
	return srv.currentServices().scoreboardService
}

func (srv *Server) getSubmissionService() *services.SubmissionService {
	return srv.currentServices().submissionService
}

func (srv *Server) getSystemSettingService() *services.SystemSettingService {
	return srv.currentServices().systemSettingService
}

func (srv *Server) getTeamService() *services.TeamService {
	return srv.currentServices().teamService
}

func (srv *Server) getTemplateRuntimeService() *services.TemplateRuntimeService {
	return srv.currentServices().templateRuntimeService
}

func (srv *Server) getTheoryService() *services.TheoryService {
	return srv.currentServices().theoryService
}

func (srv *Server) getTheorySubmissionService() *services.TheorySubmissionService {
	return srv.currentServices().theorySubmissionService
}

func (srv *Server) getUserService() *services.UserService {
	return srv.currentServices().userService
}

func (srv *Server) getVirtualizationHostService() *services.VirtualizationHostService {
	return srv.currentServices().virtualizationHostService
}

func (srv *Server) getVulnLabService() *services.VulnLabService {
	return srv.currentServices().vulnLabService
}

func (srv *Server) getWriteupService() *services.WriteupService {
	return srv.currentServices().writeupService
}
