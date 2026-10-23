package httpserver

import (
	"net/http"

	"cloudctf/backend/internal/httpserver/response"
)

func (srv *Server) registerAPIRoutes(mux *http.ServeMux) {
	srv.registerPrimaryAPIRoutes(mux)
}

func (srv *Server) registerPrimaryAPIRoutes(mux *http.ServeMux) {
	srv.registerAdminRoutes(mux)
	srv.registerAdminResourceRoutes(mux)
	srv.registerPlayerRoutes(mux)
	srv.registerPublicRoutes(mux)
	mux.HandleFunc("POST /mcp", srv.mcpHandler)
}

func (srv *Server) registerPublicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/public/settings", srv.publicSettingsHandler)
	mux.HandleFunc("GET /api/public/version", srv.publicVersionHandler)
	mux.HandleFunc("/api/public/", apiRouteFallback("public"))
}

func (srv *Server) registerPlayerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/login", srv.playerLoginHandler)
	mux.HandleFunc("POST /api/register", srv.playerRealRegisterHandler)
	mux.HandleFunc("GET /api/me", srv.playerMeHandler)
	mux.HandleFunc("POST /api/refresh", srv.playerRefreshHandler)
	mux.HandleFunc("POST /api/logout", srv.playerLogoutHandler)
	mux.HandleFunc("POST /api/stream-ticket", srv.streamTicketHandler)
	mux.HandleFunc("GET /api/user/api-key", srv.playerAPIKeyStatusHandler)
	mux.HandleFunc("POST /api/user/api-key", srv.playerAPIKeyGenerateHandler)
	mux.HandleFunc("POST /api/user/api-key/reset", srv.playerAPIKeyResetHandler)
	mux.HandleFunc("GET /api/mcp/config", srv.playerMCPConfigHandler)

	mux.HandleFunc("GET /api/competitions", srv.playerCompetitionsHandler)
	mux.HandleFunc("GET /api/competitions/{id}", srv.playerCompetitionDetailHandler)
	mux.HandleFunc("POST /api/competitions/{id}/register", srv.playerRegisterCompetitionHandler)
	mux.HandleFunc("GET /api/competitions/{id}/announcements", srv.playerCompetitionAnnouncementsHandler)
	mux.HandleFunc("GET /api/competitions/{id}/scoreboard", srv.playerCompetitionScoreboardHandler)
	mux.HandleFunc("GET /api/competitions/{id}/exam", srv.playerCompetitionExamHandler)
	mux.HandleFunc("GET /api/competitions/{id}/writeup", srv.playerWriteupHandler)
	mux.HandleFunc("POST /api/competitions/{id}/writeup", srv.playerWriteupHandler)

	mux.HandleFunc("GET /api/practice/challenges", srv.playerPracticeChallengesHandler)
	mux.HandleFunc("GET /api/challenges/{id}", srv.playerChallengeDetailHandler)
	mux.HandleFunc("POST /api/challenges/{id}/submit", srv.playerFlagSubmitHandler)
	mux.HandleFunc("POST /api/challenges/{id}/start", srv.playerStartContainerHandler)
	mux.HandleFunc("POST /api/challenges/{id}/extend", srv.playerExtendContainerHandler)
	mux.HandleFunc("POST /api/challenges/{id}/stop", srv.playerStopContainerHandler)
	mux.HandleFunc("POST /api/challenges/{id}/hints/{hint_index}/purchase", srv.playerPurchaseHintHandler)

	mux.HandleFunc("GET /api/user/stats", srv.playerStatsHandler)

	mux.HandleFunc("GET /api/leaderboard/practice", srv.playerPracticeLeaderboardHandler)
	mux.HandleFunc("GET /api/leaderboard/challenges", srv.playerChallengeLeaderboardHandler)
	mux.HandleFunc("GET /api/leaderboard/my-rank", srv.playerMyRankHandler)

	mux.HandleFunc("GET /api/team/my", srv.playerTeamHandler)
	mux.HandleFunc("POST /api/team", srv.playerCreateTeamHandler)
	mux.HandleFunc("POST /api/team/join", srv.playerJoinTeamHandler)
	mux.HandleFunc("POST /api/team/leave", srv.playerLeaveTeamHandler)
	mux.HandleFunc("GET /api/teams", srv.playerTeamsListHandler)

	mux.HandleFunc("GET /api/vuln-labs", srv.playerVulnLabsHandler)
	mux.HandleFunc("GET /api/vuln-labs/{id}", srv.playerVulnLabDetailHandler)
	mux.HandleFunc("POST /api/vuln-labs/{id}/start", srv.playerVulnLabStartHandler)
	mux.HandleFunc("POST /api/vuln-labs/{id}/extend", srv.playerVulnLabExtendHandler)
	mux.HandleFunc("POST /api/vuln-labs/{id}/stop", srv.playerVulnLabStopHandler)
	mux.HandleFunc("GET /api/vuln-labs/{id}/shell", srv.playerVulnLabShellWebsocketHandler)

	mux.HandleFunc("GET /api/notifications", srv.playerNotificationsInboxHandler)
	mux.HandleFunc("PUT /api/notifications/{id}/read", srv.adminMarkOneReadHandler)
	mux.HandleFunc("PUT /api/notifications/mark-all-read", srv.adminMarkAllNotificationsReadHandler)

	mux.HandleFunc("GET /api/exam/{id}", srv.playerExamGetHandler)
	mux.HandleFunc("POST /api/exam/{id}/submit", srv.playerExamSubmitHandler)

	mux.HandleFunc("/api/", apiRouteFallback("player"))
}

func (srv *Server) registerAdminRoutes(mux *http.ServeMux) {
	// Auth
	mux.HandleFunc("POST /api-admin/auth/login", srv.adminLoginHandler)
	mux.HandleFunc("GET /api-admin/auth/me", srv.adminMeHandler)
	mux.HandleFunc("POST /api-admin/auth/refresh", srv.adminRefreshHandler)
	mux.HandleFunc("POST /api-admin/auth/logout", srv.adminLogoutHandler)
	mux.HandleFunc("POST /api-admin/auth/change-password", srv.adminChangePasswordHandler)
	mux.HandleFunc("POST /api-admin/auth/stream-ticket", srv.streamTicketHandler)

	// Dashboard
	mux.HandleFunc("GET /api-admin/dashboard/stats", srv.adminDashboardFullStatsHandler)
	mux.HandleFunc("GET /api-admin/dashboard/recent-competitions", srv.adminRecentCompetitionsHandler)
	mux.HandleFunc("GET /api-admin/dashboard/hot-challenges", srv.adminHotChallengesHandler)
	mux.HandleFunc("GET /api-admin/dashboard/active-users", srv.adminActiveUsersHandler)
	mux.HandleFunc("GET /api-admin/dashboard/submission-trend", srv.adminSubmissionTrendHandler)
	mux.HandleFunc("GET /api-admin/dashboard/user-growth", srv.adminUserGrowthHandler)

	// Users
	mux.HandleFunc("GET /api-admin/users/template", adminUsersTemplateHandler)
	mux.HandleFunc("POST /api-admin/users/batch-delete", srv.adminBatchDeleteUsersHandler)
	mux.HandleFunc("POST /api-admin/users/batch/delete", srv.adminBatchDeleteUsersHandler)
	mux.HandleFunc("POST /api-admin/users/batch-reset-password", srv.adminBatchResetUserPasswordsHandler)
	mux.HandleFunc("POST /api-admin/users/batch-ban", srv.adminBatchBanUsersHandler)
	mux.HandleFunc("POST /api-admin/users/batch/ban", srv.adminBatchBanUsersHandler)
	mux.HandleFunc("POST /api-admin/users/batch-unban", srv.adminBatchUnbanUsersHandler)
	mux.HandleFunc("POST /api-admin/users/batch/unban", srv.adminBatchUnbanUsersHandler)
	mux.HandleFunc("POST /api-admin/users/tags", srv.adminUserTagsMutationHandler)
	mux.HandleFunc("GET /api-admin/users/search", srv.adminUserSearchHandler)
	mux.HandleFunc("POST /api-admin/users/{id}/ban", srv.adminBanUserHandler)
	mux.HandleFunc("PATCH /api-admin/users/{id}/disable", srv.adminUserDisableHandler)
	mux.HandleFunc("POST /api-admin/users/{id}/unban", srv.adminUnbanUserHandler)
	mux.HandleFunc("PATCH /api-admin/users/{id}/enable", srv.adminUserEnableHandler)
	mux.HandleFunc("POST /api-admin/users/{id}/reset-password", srv.adminResetUserPasswordHandler)
	mux.HandleFunc("PUT /api-admin/users/{id}/tags", srv.adminUserTagsMutationHandler)

	// User management
	mux.HandleFunc("POST /api-admin/user-management/users/import", adminUserImportHandler)
	mux.HandleFunc("GET /api-admin/user-management/users/export-template", adminUsersTemplateHandler)
	mux.HandleFunc("POST /api-admin/user-management/users/tags", srv.adminUserTagsMutationHandler)
	mux.HandleFunc("DELETE /api-admin/user-management/users/tags", srv.adminUserTagsMutationHandler)
	mux.HandleFunc("POST /api-admin/user-management/users/reset-password", srv.adminBatchResetUserPasswordsHandler)

	// Teams
	mux.HandleFunc("GET /api-admin/teams/template", adminTeamsTemplateHandler)
	mux.HandleFunc("POST /api-admin/teams/import", srv.adminTeamImportHandler)
	mux.HandleFunc("GET /api-admin/teams/export", srv.adminTeamExportHandler)
	mux.HandleFunc("POST /api-admin/teams/{id}/blacklist", srv.adminTeamBlacklistHandler)
	mux.HandleFunc("DELETE /api-admin/teams/{id}/blacklist", srv.adminTeamUnblacklistHandler)
	mux.HandleFunc("POST /api-admin/teams/{id}/members", srv.adminTeamAddMemberHandler)
	mux.HandleFunc("DELETE /api-admin/teams/{id}/members/{user_id}", srv.adminTeamRemoveMemberHandler)
	mux.HandleFunc("GET /api-admin/teams/{id}/member-history", srv.adminTeamMemberHistoryHandler)
	mux.HandleFunc("GET /api-admin/teams/users/{user_id}/team-history", srv.adminUserTeamHistoryHandler)

	// Challenges
	mux.HandleFunc("POST /api-admin/challenges/{id}/submit", srv.adminFlagSubmitHandler)
	mux.HandleFunc("POST /api-admin/challenges/{id}/start", srv.adminChallengeStartContainerHandler)
	mux.HandleFunc("POST /api-admin/challenges/{id}/stop", adminChallengeStopContainerHandler)
	mux.HandleFunc("POST /api-admin/challenges/import", srv.adminChallengeImportHandler)
	mux.HandleFunc("GET /api-admin/challenges/export", srv.adminChallengeExportHandler)
	mux.HandleFunc("POST /api-admin/challenges/{id}/files", srv.adminUploadHandler)
	mux.HandleFunc("DELETE /api-admin/challenges/{id}/files/{file_id}", srv.adminChallengeFileDeleteHandler)
	mux.HandleFunc("POST /api-admin/challenges/{id}/hints", srv.adminChallengeHintCreateHandler)
	mux.HandleFunc("DELETE /api-admin/challenges/{id}/hints/{hint_id}", srv.adminChallengeHintDeleteHandler)

	// Flags
	mux.HandleFunc("GET /api-admin/flags/dynamic/{id}", srv.adminDynamicFlagHandler)
	mux.HandleFunc("GET /api-admin/flags/view/{id}", srv.adminViewDynamicFlagHandler)

	// Theory
	mux.HandleFunc("POST /api-admin/theory/import", srv.adminTheoryImportHandler)
	mux.HandleFunc("GET /api-admin/theory/export", srv.adminTheoryExportHandler)
	mux.HandleFunc("POST /api-admin/theory/{id}/submit", adminTheorySubmitHandler)

	// Exam
	mux.HandleFunc("POST /api-admin/exam/quick-create", srv.adminExamQuickCreateHandler)
	mux.HandleFunc("POST /api-admin/exam/create", srv.adminExamCreateHandler)
	mux.HandleFunc("GET /api-admin/exam/statistics/{id}", srv.adminExamStatisticsHandler)

	srv.registerAdminCompetitionRoutes(mux)

	// Scoreboard
	mux.HandleFunc("POST /api-admin/scoreboard/{id}/freeze", srv.adminFreezeScoreboardHandler)
	mux.HandleFunc("POST /api-admin/scoreboard/{id}/unfreeze", srv.adminUnfreezeScoreboardHandler)
	mux.HandleFunc("GET /api-admin/scoreboard/{id}/statistics", srv.adminScoreboardStatisticsHandler)
	mux.HandleFunc("GET /api-admin/scoreboard/{id}/solve-status", srv.adminSolveStatusHandler)

	srv.registerAdminNotificationRoutes(mux)

	// Announcements
	mux.HandleFunc("PUT /api-admin/announcements/{id}/publish", srv.adminPublishAnnouncementHandler)
	mux.HandleFunc("PUT /api-admin/announcements/{id}/unpublish", srv.adminUnpublishAnnouncementHandler)
	mux.HandleFunc("PUT /api-admin/announcements/{id}/pin", srv.adminPinAnnouncementHandler)
	mux.HandleFunc("PUT /api-admin/announcements/{id}/unpin", srv.adminUnpinAnnouncementHandler)

	// Audit
	mux.HandleFunc("GET /api-admin/audit/logs/export", srv.adminAuditExportHandler)

	srv.registerAdminSettingsRoutes(mux)

	// Resource repositories
	mux.HandleFunc("POST /api-admin/resource-repositories/{id}/sync", srv.adminResourceRepoSyncHandler)
	mux.HandleFunc("POST /api-admin/resource-repositories/sync", srv.adminResourceRepoSyncHandler)

	srv.registerAdminVirtualizationRoutes(mux)

	// Roles
	mux.HandleFunc("GET /api-admin/roles/permissions", adminAllPermissionsHandler)
	mux.HandleFunc("GET /api-admin/roles/{id}/permissions", srv.adminRolePermissionsHandler)

	// Upload
	mux.HandleFunc("/api-admin/upload/", srv.adminUploadHandler)

	// Explicit resource endpoints
	mux.HandleFunc("GET /api-admin/referral-domain-stats", adminReferralDomainStatsHandler)
	mux.HandleFunc("GET /api-admin/referral-domain-stats/", adminReferralDomainStatsHandler)
	mux.HandleFunc("GET /api-admin/resource-permissions", adminResourcePermissionsHandler)
	mux.HandleFunc("GET /api-admin/resource-permissions/", adminResourcePermissionsHandler)

	mux.HandleFunc("/api-admin/", apiRouteFallback("admin"))
}

func (srv *Server) registerAdminResourceRoutes(mux *http.ServeMux) {
	registerCompatCollectionRoute := func(path string, get, post http.HandlerFunc) {
		if get != nil {
			mux.HandleFunc("GET "+path, get)
		}
		if post != nil {
			mux.HandleFunc("POST "+path, post)
		}
	}

	registerCompatCollectionRoute("/api-admin/resource-repositories", srv.adminResourceRepositoriesHandler, srv.adminResourceRepositoryCreateHandler)
	mux.HandleFunc("GET /api-admin/resource-repositories/{id}", srv.adminResourceRepositoryDetailHandler)
	mux.HandleFunc("PUT /api-admin/resource-repositories/{id}", srv.adminResourceRepositoryUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/resource-repositories/{id}", srv.adminResourceRepositoryDeleteHandler)

	registerCompatCollectionRoute("/api-admin/vuln-labs", srv.adminVulnLabsHandler, srv.adminVulnLabCreateHandler)
	mux.HandleFunc("GET /api-admin/vuln-labs/{id}", srv.adminVulnLabDetailHandler)
	mux.HandleFunc("PUT /api-admin/vuln-labs/{id}", srv.adminVulnLabUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/vuln-labs/{id}", srv.adminVulnLabDeleteHandler)

	registerCompatCollectionRoute("/api-admin/announcements", srv.adminAnnouncementsHandler, srv.adminAnnouncementCreateHandler)
	mux.HandleFunc("GET /api-admin/announcements/{id}", srv.adminAnnouncementDetailHandler)
	mux.HandleFunc("PUT /api-admin/announcements/{id}", srv.adminAnnouncementUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/announcements/{id}", srv.adminAnnouncementDeleteHandler)

	registerCompatCollectionRoute("/api-admin/teams", srv.adminTeamsHandler, srv.adminTeamCreateHandler)
	mux.HandleFunc("GET /api-admin/teams/{id}", srv.adminTeamDetailHandler)
	mux.HandleFunc("PUT /api-admin/teams/{id}", srv.adminTeamUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/teams/{id}", srv.adminTeamDeleteHandler)

	registerCompatCollectionRoute("/api-admin/users", srv.adminUsersHandler("users"), srv.adminUserCreateHandler("users"))
	mux.HandleFunc("GET /api-admin/users/{id}", srv.adminUserDetailHandler)
	mux.HandleFunc("PUT /api-admin/users/{id}", srv.adminUserUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/users/{id}", srv.adminUserDeleteHandler)

	registerCompatCollectionRoute("/api-admin/admins", srv.adminUsersHandler("admins"), srv.adminUserCreateHandler("admins"))
	mux.HandleFunc("GET /api-admin/admins/{id}", srv.adminUserDetailHandler)
	mux.HandleFunc("PUT /api-admin/admins/{id}", srv.adminUserUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/admins/{id}", srv.adminUserDeleteHandler)

	registerCompatCollectionRoute("/api-admin/roles", srv.adminRolesHandler, srv.adminRoleCreateHandler)
	mux.HandleFunc("PUT /api-admin/roles/{id}", srv.adminRoleUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/roles/{id}", srv.adminRoleDeleteHandler)

	registerCompatCollectionRoute("/api-admin/challenges", srv.adminChallengesHandler, srv.adminChallengeCreateHandler)
	mux.HandleFunc("GET /api-admin/challenges/{id}", srv.adminChallengeDetailHandler)
	mux.HandleFunc("PUT /api-admin/challenges/{id}", srv.adminChallengeUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/challenges/{id}", srv.adminChallengeDeleteHandler)

	registerCompatCollectionRoute("/api-admin/theory/questions", srv.adminTheoryQuestionsHandler, srv.adminTheoryQuestionCreateHandler)
	mux.HandleFunc("GET /api-admin/theory/questions/{id}", srv.adminTheoryQuestionDetailHandler)
	mux.HandleFunc("PUT /api-admin/theory/questions/{id}", srv.adminTheoryQuestionUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/theory/questions/{id}", srv.adminTheoryQuestionDeleteHandler)
	mux.HandleFunc("POST /api-admin/theory/questions/import", srv.adminTheoryImportHandler)
	mux.HandleFunc("GET /api-admin/theory/questions/export", srv.adminTheoryExportHandler)
	mux.HandleFunc("GET /api-admin/theory/question/stats", srv.adminQuestionStatsHandler)

	registerCompatCollectionRoute("/api-admin/exam", srv.adminExamsHandler, srv.adminExamResourceCreateHandler)
	mux.HandleFunc("PUT /api-admin/exam/{id}", srv.adminExamResourceUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/exam/{id}", srv.adminExamResourceDeleteHandler)
	mux.HandleFunc("GET /api-admin/exam/submissions", srv.adminExamSubmissionsHandler)
	mux.HandleFunc("GET /api-admin/exam/submissions/{id}", srv.adminExamSubmissionDetailHandler)
	mux.HandleFunc("GET /api-admin/exam/competitions/{id}/exam", srv.adminCompetitionExamHandler)
	mux.HandleFunc("POST /api-admin/exam/competitions/{id}/exam", srv.adminCompetitionExamHandler)
	mux.HandleFunc("POST /api-admin/exam/competitions/{id}/exam/quick-create", srv.adminExamQuickCreateHandler)
	mux.HandleFunc("POST /api-admin/exam/competitions/{id}/exam/bind", srv.adminCompetitionExamBindHandler)
	mux.HandleFunc("PUT /api-admin/exam/exams/{id}", srv.adminUpdateCompetitionExamHandler)
	mux.HandleFunc("DELETE /api-admin/exam/competitions/{id}/exam", srv.adminCompetitionExamHandler)

	registerCompatCollectionRoute("/api-admin/submissions", srv.adminSubmissionsHandler, srv.adminSubmissionCreateHandler)
	mux.HandleFunc("DELETE /api-admin/submissions/{id}", srv.adminSubmissionDeleteHandler)
	mux.HandleFunc("GET /api-admin/audit/logs", srv.adminAuditLogsHandler)
}

func (srv *Server) registerAdminCompetitionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PATCH /api-admin/competitions/{id}/status", srv.adminCompetitionStatusHandler)
	mux.HandleFunc("GET /api-admin/competitions/{id}/admins", srv.adminCompetitionAdminsHandler)

	mux.HandleFunc("GET /api-admin/competitions/{id}/participants", srv.adminCompetitionParticipantsHandler)
	mux.HandleFunc("POST /api-admin/competitions/{id}/participants", srv.adminAddParticipantHandler)
	mux.HandleFunc("DELETE /api-admin/competitions/{id}/participants/{user_id}", srv.adminRemoveParticipantHandler)
	mux.HandleFunc("POST /api-admin/competitions/{id}/participants/generate", srv.adminGenerateCompetitionParticipantsHandler)

	mux.HandleFunc("GET /api-admin/competitions/{id}/challenges", srv.adminCompetitionChallengesHandler)
	mux.HandleFunc("POST /api-admin/competitions/{id}/challenges", srv.adminAddChallengeToCompetitionHandler)
	mux.HandleFunc("DELETE /api-admin/competitions/{id}/challenges/{challenge_id}", srv.adminRemoveChallengeFromCompetitionHandler)
	mux.HandleFunc("GET /api-admin/theory/competitions/{id}/questions", srv.adminCompetitionTheoryQuestionsHandler)
	mux.HandleFunc("POST /api-admin/theory/competitions/{id}/questions", srv.adminAddTheoryQuestionToCompetitionHandler)
	mux.HandleFunc("DELETE /api-admin/theory/competitions/{id}/questions/{qid}", srv.adminRemoveTheoryQuestionFromCompetitionHandler)

	mux.HandleFunc("GET /api-admin/competitions/{id}/announcements", srv.playerCompetitionAnnouncementsHandler)
	mux.HandleFunc("GET /api-admin/competitions/{id}/submissions", srv.adminCompetitionSubmissionsHandler)
	mux.HandleFunc("GET /api-admin/competitions/{id}/writeups", srv.adminListWriteupsHandler)
	mux.HandleFunc("DELETE /api-admin/writeups/{id}", srv.adminDeleteWriteupHandler)

	mux.HandleFunc("GET /api-admin/competitions", srv.adminCompetitionsHandler)
	mux.HandleFunc("POST /api-admin/competitions", srv.adminCompetitionCreateHandler)
	mux.HandleFunc("GET /api-admin/competitions/{id}", srv.adminCompetitionDetailHandler)
	mux.HandleFunc("PUT /api-admin/competitions/{id}", srv.adminCompetitionUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/competitions/{id}", srv.adminCompetitionDeleteHandler)
}

func (srv *Server) registerAdminNotificationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api-admin/notifications", srv.adminNotificationsInboxHandler)
	mux.HandleFunc("GET /api-admin/notifications/unread-count", srv.adminUnreadCountHandler)
	mux.HandleFunc("PUT /api-admin/notifications/mark-all-read", srv.adminMarkAllNotificationsReadHandler)
	mux.HandleFunc("PUT /api-admin/notifications/{id}/read", srv.adminMarkOneReadHandler)

	mux.HandleFunc("POST /api-admin/notifications", srv.adminNotificationCreateHandler)
	mux.HandleFunc("PUT /api-admin/notifications/{id}", srv.adminNotificationUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/notifications/{id}", srv.adminNotificationDeleteHandler)
}

func (srv *Server) registerAdminSettingsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api-admin/settings/", srv.adminSettingsHandler)
	mux.HandleFunc("/api-admin/settings", srv.adminSettingsHandler)

	mux.HandleFunc("GET /api-admin/settings/settings", srv.adminSettingsHandler)
	mux.HandleFunc("PUT /api-admin/settings/settings", srv.adminSettingsHandler)
	mux.HandleFunc("GET /api-admin/settings/settings/info", srv.settingsInfoHandler)
	mux.HandleFunc("GET /api-admin/settings/update/check", srv.adminUpdateCheckHandler)
	mux.HandleFunc("POST /api-admin/settings/update/apply", srv.adminUpdateApplyHandler)
	mux.HandleFunc("POST /api-admin/settings/update/upload", srv.adminUpdateUploadHandler)
	mux.HandleFunc("GET /api-admin/settings/update/status", srv.adminUpdateStatusHandler)
	mux.HandleFunc("POST /api-admin/settings/cache/clear", adminSettingsCacheClearHandler)
	mux.HandleFunc("POST /api-admin/settings/demo-data/generate", srv.adminGenerateDemoDataHandler)
	mux.HandleFunc("GET /api-admin/settings/logs", srv.adminSettingsLogsHandler)

	mux.HandleFunc("POST /api-admin/settings/backup", srv.adminSettingsBackupCreateHandler)
	mux.HandleFunc("GET /api-admin/settings/backups", srv.adminSettingsBackupListHandler)
	mux.HandleFunc("POST /api-admin/settings/restore/{id}", srv.adminSettingsBackupRestoreHandler)
	mux.HandleFunc("DELETE /api-admin/settings/backups/{id}", srv.adminSettingsBackupDeleteHandler)
	mux.HandleFunc("GET /api-admin/settings/backups/{id}/download", srv.adminSettingsBackupDownloadHandler)
	mux.HandleFunc("POST /api-admin/settings/backups/upload", srv.adminSettingsBackupUploadHandler)
}

func (srv *Server) registerAdminVirtualizationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api-admin/virtualization/hosts", srv.adminVirtualizationHostsHandler)
	mux.HandleFunc("POST /api-admin/virtualization/hosts", srv.adminVirtualizationHostCreateHandler)
	mux.HandleFunc("GET /api-admin/virtualization/hosts/{id}", srv.adminVirtualizationHostDetailHandler)
	mux.HandleFunc("PUT /api-admin/virtualization/hosts/{id}", srv.adminVirtualizationHostUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/virtualization/hosts/{id}", srv.adminVirtualizationHostDeleteHandler)
	mux.HandleFunc("GET /api-admin/virtualization/hosts/{id}/check", srv.adminVirtualizationHostCheckHandler)
	mux.HandleFunc("GET /api-admin/virtualization/hosts/{id}/images", srv.adminVirtualizationHostImagesHandler)
	mux.HandleFunc("POST /api-admin/virtualization/hosts/{id}/images/pull", srv.adminVirtualizationHostImagePullHandler)
	mux.HandleFunc("POST /api-admin/virtualization/hosts/{id}/images/build", srv.adminVirtualizationHostImageBuildHandler)
	mux.HandleFunc("POST /api-admin/virtualization/hosts/{id}/images/load", srv.adminVirtualizationHostImageLoadHandler)

	mux.HandleFunc("GET /api-admin/virtualization/containers", srv.adminVirtualizationContainersHandler)
	mux.HandleFunc("POST /api-admin/virtualization/containers", srv.adminVirtualizationContainerCreateHandler)
	mux.HandleFunc("GET /api-admin/virtualization/containers/{id}", srv.adminVirtualizationContainerDetailHandler)
	mux.HandleFunc("PUT /api-admin/virtualization/containers/{id}", srv.adminVirtualizationContainerUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/virtualization/containers/{id}", srv.adminVirtualizationContainerDeleteHandler)
	mux.HandleFunc("POST /api-admin/virtualization/containers/action", srv.adminVirtualizationContainerActionHandler)
	mux.HandleFunc("POST /api-admin/virtualization/containers/batch-action", srv.adminVirtualizationContainerBatchActionHandler)

	mux.HandleFunc("GET /api-admin/virtualization/templates", srv.adminVirtualizationTemplatesHandler)
	mux.HandleFunc("GET /api-admin/virtualization/templates/options", srv.adminVirtualizationTemplatesHandler)
	mux.HandleFunc("POST /api-admin/virtualization/templates", srv.adminVirtualizationTemplateCreateHandler)
	mux.HandleFunc("GET /api-admin/virtualization/templates/{id}", srv.adminVirtualizationTemplateDetailHandler)
	mux.HandleFunc("PUT /api-admin/virtualization/templates/{id}", srv.adminVirtualizationTemplateUpdateHandler)
	mux.HandleFunc("DELETE /api-admin/virtualization/templates/{id}", srv.adminVirtualizationTemplateDeleteHandler)
	mux.HandleFunc("POST /api-admin/virtualization/templates/batch/delete", srv.adminVirtualizationTemplateBatchDeleteHandler)
	mux.HandleFunc("POST /api-admin/virtualization/templates/batch/build", srv.adminVirtualizationTemplateBatchBuildHandler)
	mux.HandleFunc("POST /api-admin/virtualization/templates/{id}/start", srv.adminVirtualizationTemplateStartHandler)
	mux.HandleFunc("POST /api-admin/virtualization/templates/{id}/build", srv.adminVirtualizationTemplateBuildHandler)
	mux.HandleFunc("GET /api-admin/virtualization/templates/{id}/build/stream", adminVirtualizationTemplateBuildStreamHandler)

	mux.HandleFunc("GET /api-admin/virtualization/container-groups", srv.adminVirtualizationContainerGroupsHandler)
	mux.HandleFunc("POST /api-admin/virtualization/container-groups", srv.adminVirtualizationContainerGroupCreateHandler)
	mux.HandleFunc("GET /api-admin/virtualization/container-groups/{id}", srv.adminVirtualizationContainerGroupDetailHandler)
	mux.HandleFunc("DELETE /api-admin/virtualization/container-groups/{id}", srv.adminVirtualizationContainerGroupDeleteHandler)
	mux.HandleFunc("POST /api-admin/virtualization/container-groups/{id}/start", srv.adminVirtualizationContainerGroupStartHandler)
	mux.HandleFunc("POST /api-admin/virtualization/container-groups/{id}/stop", srv.adminVirtualizationContainerGroupStopHandler)
}

func (srv *Server) publicSettingsHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{
		"data": srv.publicSystemSettings(),
	})
}

func (srv *Server) publicVersionHandler(w http.ResponseWriter, r *http.Request) {
	cfg := srv.currentConfig()
	response.JSON(w, http.StatusOK, map[string]any{
		"service": cfg.AppName,
		"version": srv.currentVersion(),
	})
}
