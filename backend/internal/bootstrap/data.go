package bootstrap

import (
	"cloudctf/backend/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func anyToJsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func seedRoleData(db *gorm.DB) error {
	roles := []models.Role{
		{Name: "superadmin", Description: "超级管理员", Permissions: anyToJsonString([]string{"*"}), IsSystem: true},
		{Name: "admin", Description: "管理员", Permissions: anyToJsonString([]string{"admin:*"}), IsSystem: true},
		{Name: "player", Description: "参赛用户", Permissions: anyToJsonString([]string{}), IsSystem: true},
	}
	for _, role := range roles {
		var existing models.Role
		err := db.Where("name = ?", role.Name).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&role).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedUserData(db *gorm.DB) error {
	var roles []models.Role
	if err := db.Find(&roles).Error; err != nil {
		return err
	}

	roleIDs := make(map[string]uint, len(roles))
	for _, role := range roles {
		roleIDs[role.Name] = role.ID
	}

	getRoleID := func(roleName string) (*uint, error) {
		id, ok := roleIDs[roleName]
		if !ok || id == 0 {
			return nil, fmt.Errorf("role %q not found", roleName)
		}
		return &id, nil
	}

	superAdminRoleID, err := getRoleID("superadmin")
	if err != nil {
		return err
	}
	playerRoleID, err := getRoleID("player")
	if err != nil {
		return err
	}

	hashPassword := func(password string) (string, error) {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}
		return string(hashed), nil
	}

	adminUsername := strings.TrimSpace(os.Getenv("FIRST_SUPERUSER_USERNAME"))
	if adminUsername == "" {
		adminUsername = "admin"
	}
	adminPasswordValue := strings.TrimSpace(os.Getenv("FIRST_SUPERUSER_PASSWORD"))
	if adminPasswordValue == "" {
		adminPasswordValue = "admin123456"
	}
	adminPassword, err := hashPassword(adminPasswordValue)
	if err != nil {
		return err
	}
	playerPassword, err := hashPassword("password123")
	if err != nil {
		return err
	}

	users := []models.User{
		{
			Username:       adminUsername,
			Email:          "admin@cloudctf.local",
			HashedPassword: adminPassword,
			FullName:       "超级管理员",
			RoleID:         superAdminRoleID,
			IsActive:       true,
			IsSuperuser:    true,
			Tags:           "[]",
		},
	}
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		users = append(users, models.User{
			Username:       "player1",
			Email:          "player1@example.com",
			HashedPassword: playerPassword,
			FullName:       "参赛用户1",
			RoleID:         playerRoleID,
			IsActive:       true,
			IsSuperuser:    false,
			Tags:           "[]",
		})
	}

	for _, user := range users {
		var existing models.User
		err := db.Where("username = ? OR email = ?", user.Username, user.Email).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}

	return nil
}

func defaultDockerEndpoint() string {
	if host := strings.TrimSpace(os.Getenv("DOCKER_HOST")); host != "" {
		return host
	}
	if runtime.GOOS == "windows" {
		return "npipe:////./pipe/docker_engine"
	}
	return "unix:///var/run/docker.sock"
}

func seedResourceRepoData(db *gorm.DB) error {
	var virCount int64
	if err := db.Model(&models.VirtualizationHost{}).Count(&virCount).Error; err != nil {
		return err
	}
	if virCount == 0 {
		host := models.VirtualizationHost{
			Name:      "Local Docker",
			HostType:  "docker",
			IsDefault: true,
			Endpoint:  defaultDockerEndpoint(),
			Status:    "online",
			IsActive:  true,
			Config:    `{"max_containers":100,"port_range_start":30000,"port_range_end":40000}`,
		}
		if err := db.Create(&host).Error; err != nil {
			return err
		}
	}

	var vulnCount int64
	if err := db.Model(&models.VulnLab{}).Count(&vulnCount).Error; err != nil {
		return err
	}
	var repoCount int64
	if err := db.Model(&models.ResourceRepo{}).Count(&repoCount).Error; err != nil {
		return err
	}
	if repoCount == 0 {
		repos := []models.ResourceRepo{
			{
				Name:             "Vulhub",
				RepoType:         "vul",
				Provider:         "git",
				RepositoryURL:    "https://github.com/vulhub/vulhub.git",
				RepositoryBranch: "master",
				TemplatesPath:    "",
				Enabled:          true,
				Tags:             `["vulhub"]`,
			},
			{
				Name:             "CTFDB",
				RepoType:         "ctf",
				Provider:         "git",
				RepositoryURL:    "https://github.com/hexbay/ctfdb.git",
				RepositoryBranch: "main",
				TemplatesPath:    "",
				Enabled:          true,
				Tags:             `["ctfdb"]`,
			},
		}
		for _, repo := range repos {

			if err := db.Create(&repo).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
