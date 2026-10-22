package services

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cloudctf/backend/internal/config"
	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type BackupManifest struct {
	Version        string         `json:"version"`
	AppName        string         `json:"app_name"`
	CreatedAt      string         `json:"created_at"`
	CreatedBy      string         `json:"created_by"`
	DatabaseDriver string         `json:"database_driver"`
	UploadDir      string         `json:"upload_dir"`
	Counts         map[string]int `json:"counts"`
	Checksum       string         `json:"checksum"`
}

type BackupFileInfo struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

type backupPayload struct {
	Manifest BackupManifest             `json:"manifest"`
	Data     map[string]json.RawMessage `json:"data"`
}

type BackupService struct {
	db *gorm.DB
}

func NewBackupService(db *gorm.DB) *BackupService {
	return &BackupService{db: db}
}

func (s *BackupService) ListFiles(dir string) ([]BackupFileInfo, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	items := make([]BackupFileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, BackupFileInfo{
			Name:      entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format(time.RFC3339),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	return items, nil
}

func (s *BackupService) CreateArchive(cfg config.Config, createdBy string) (BackupFileInfo, error) {
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		return BackupFileInfo{}, err
	}
	snapshot, counts, err := s.collectData()
	if err != nil {
		return BackupFileInfo{}, err
	}
	driver := "unknown"
	if s.db != nil && s.db.Dialector != nil {
		driver = s.db.Dialector.Name()
	}
	manifest := BackupManifest{
		Version:        "v1",
		AppName:        cfg.AppName,
		CreatedAt:      time.Now().Format(time.RFC3339),
		CreatedBy:      createdBy,
		DatabaseDriver: driver,
		UploadDir:      cfg.UploadDir,
		Counts:         counts,
	}
	payload := backupPayload{Manifest: manifest, Data: snapshot}
	dataBytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return BackupFileInfo{}, err
	}
	sum := sha256.Sum256(dataBytes)
	payload.Manifest.Checksum = hex.EncodeToString(sum[:])
	dataBytes, err = json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return BackupFileInfo{}, err
	}

	name := fmt.Sprintf("cloudctf-backup-%s.zip", time.Now().Format("20060102-150405"))
	tmpPath := filepath.Join(cfg.BackupDir, name+".tmp")
	finalPath := filepath.Join(cfg.BackupDir, name)
	if err := writeBackupZip(tmpPath, dataBytes, cfg.UploadDir); err != nil {
		return BackupFileInfo{}, err
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return BackupFileInfo{}, err
	}
	_ = s.pruneOldBackups(cfg.BackupDir, cfg.BackupRetention)
	fileInfo, err := os.Stat(finalPath)
	if err != nil {
		return BackupFileInfo{}, err
	}
	return BackupFileInfo{Name: name, Size: fileInfo.Size(), CreatedAt: fileInfo.ModTime().Format(time.RFC3339)}, nil
}

func (s *BackupService) RestoreArchive(cfg config.Config, archivePath, restoredBy string) (BackupManifest, error) {
	payload, uploadDir, err := readBackupZip(archivePath)
	if err != nil {
		return BackupManifest{}, err
	}
	if payload.Manifest.Version != "v1" {
		return BackupManifest{}, fmt.Errorf("unsupported backup version %q", payload.Manifest.Version)
	}
	checksum := payload.Manifest.Checksum
	payload.Manifest.Checksum = ""
	compareBytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return BackupManifest{}, err
	}
	sum := sha256.Sum256(compareBytes)
	if checksum != "" && checksum != hex.EncodeToString(sum[:]) {
		return BackupManifest{}, errors.New("backup checksum validation failed")
	}

	_, err = s.CreateArchive(cfg, restoredBy+"-pre-restore")
	if err != nil {
		return BackupManifest{}, fmt.Errorf("pre-restore backup failed: %w", err)
	}
	if err := s.applyData(payload.Data); err != nil {
		return BackupManifest{}, err
	}
	if uploadDir != "" {
		if err := replaceDirectory(uploadDir, cfg.UploadDir); err != nil {
			return BackupManifest{}, err
		}
	}
	return payload.Manifest, nil
}

func (s *BackupService) pruneOldBackups(dir string, retain int) error {
	if retain <= 0 {
		return nil
	}
	items, err := s.ListFiles(dir)
	if err != nil {
		return err
	}
	if len(items) <= retain {
		return nil
	}
	for _, item := range items[retain:] {
		_ = os.Remove(filepath.Join(dir, item.Name))
	}
	return nil
}

func (s *BackupService) collectData() (map[string]json.RawMessage, map[string]int, error) {
	tables := map[string]any{
		"roles":                        &[]models.Role{},
		"users":                        &[]models.User{},
		"competitions":                 &[]models.Competition{},
		"teams":                        &[]models.Team{},
		"team_members":                 &[]models.TeamMember{},
		"challenges":                   &[]models.Challenge{},
		"theory_questions":             &[]models.TheoryQuestion{},
		"exams":                        &[]models.Exam{},
		"competition_theory_questions": &[]models.CompetitionTheoryQuestion{},
		"submissions":                  &[]models.Submission{},
		"hint_purchases":               &[]models.HintPurchase{},
		"scoreboards":                  &[]models.Scoreboard{},
		"notifications":                &[]models.Notification{},
		"announcements":                &[]models.Announcement{},
		"audit_logs":                   &[]models.AuditLog{},
		"virtualization_hosts":         &[]models.VirtualizationHost{},
		"vuln_labs":                    &[]models.VulnLab{},
		"resource_repositories":        &[]models.ResourceRepo{},
	}
	data := make(map[string]json.RawMessage, len(tables))
	counts := make(map[string]int, len(tables))
	for name, dest := range tables {
		if err := s.db.Find(dest).Error; err != nil {
			return nil, nil, fmt.Errorf("backup table %s failed: %w", name, err)
		}
		raw, err := json.Marshal(dest)
		if err != nil {
			return nil, nil, err
		}
		data[name] = raw
		counts[name] = countJSONArray(raw)
	}
	return data, counts, nil
}

func (s *BackupService) applyData(data map[string]json.RawMessage) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := disableForeignKeys(tx); err != nil {
			return err
		}
		for _, table := range []string{
			"resource_repo", "vuln_lab", "virtualization_host",
			"audit_log", "announcement", "notification", "scoreboard",
			"hint_purchase", "submission",
			"competition_theory_question", "exam", "theory_question", "challenge",
			"team_member", "team", "competition", "user", "role",
		} {
			statement := &gorm.Statement{DB: tx}
			if err := tx.Exec("DELETE FROM " + statement.Quote(table)).Error; err != nil {
				return fmt.Errorf("clear table %s failed: %w", table, err)
			}
		}
		if err := restoreTable(tx, data["roles"], &[]models.Role{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["users"], &[]models.User{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["competitions"], &[]models.Competition{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["teams"], &[]models.Team{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["team_members"], &[]models.TeamMember{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["challenges"], &[]models.Challenge{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["theory_questions"], &[]models.TheoryQuestion{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["exams"], &[]models.Exam{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["competition_theory_questions"], &[]models.CompetitionTheoryQuestion{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["submissions"], &[]models.Submission{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["hint_purchases"], &[]models.HintPurchase{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["scoreboards"], &[]models.Scoreboard{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["notifications"], &[]models.Notification{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["announcements"], &[]models.Announcement{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["audit_logs"], &[]models.AuditLog{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["virtualization_hosts"], &[]models.VirtualizationHost{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["vuln_labs"], &[]models.VulnLab{}); err != nil {
			return err
		}
		if err := restoreTable(tx, data["resource_repositories"], &[]models.ResourceRepo{}); err != nil {
			return err
		}
		return enableForeignKeys(tx)
	})
}

func writeBackupZip(targetPath string, dataBytes []byte, uploadDir string) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	dataEntry, err := zw.Create("backup.json")
	if err != nil {
		_ = zw.Close()
		return err
	}
	if _, err := dataEntry.Write(dataBytes); err != nil {
		_ = zw.Close()
		return err
	}
	if err := addDirectoryToZip(zw, uploadDir, "uploads"); err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

func readBackupZip(archivePath string) (backupPayload, string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return backupPayload{}, "", err
	}
	defer reader.Close()
	var payload backupPayload
	restoreUploadsDir := ""
	for _, file := range reader.File {
		switch {
		case file.Name == "backup.json":
			rc, err := file.Open()
			if err != nil {
				return backupPayload{}, "", err
			}
			raw, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return backupPayload{}, "", err
			}
			if err := json.Unmarshal(raw, &payload); err != nil {
				return backupPayload{}, "", err
			}
		case strings.HasPrefix(file.Name, "uploads/"):
			if restoreUploadsDir == "" {
				restoreUploadsDir, err = os.MkdirTemp("", "cloudctf-restore-uploads-*")
				if err != nil {
					return backupPayload{}, "", err
				}
			}
			target := filepath.Join(restoreUploadsDir, strings.TrimPrefix(file.Name, "uploads/"))
			if file.FileInfo().IsDir() {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return backupPayload{}, "", err
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return backupPayload{}, "", err
			}
			rc, err := file.Open()
			if err != nil {
				return backupPayload{}, "", err
			}
			dst, err := os.Create(target)
			if err != nil {
				_ = rc.Close()
				return backupPayload{}, "", err
			}
			_, copyErr := io.Copy(dst, rc)
			_ = dst.Close()
			_ = rc.Close()
			if copyErr != nil {
				return backupPayload{}, "", copyErr
			}
		}
	}
	if payload.Manifest.Version == "" {
		return backupPayload{}, "", errors.New("backup.json missing from archive")
	}
	return payload, restoreUploadsDir, nil
}

func addDirectoryToZip(zw *zip.Writer, srcDir, prefix string) error {
	if srcDir == "" {
		return nil
	}
	info, err := os.Stat(srcDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		name := prefix
		if rel != "." {
			name += "/" + rel
		}
		if info.IsDir() {
			if rel == "." {
				return nil
			}
			_, err := zw.Create(name + "/")
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Method = zip.Deflate
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(writer, src)
		return err
	})
}

func replaceDirectory(srcDir, dstDir string) error {
	if srcDir == "" {
		return nil
	}
	_ = os.RemoveAll(dstDir + ".rollback")
	if _, err := os.Stat(dstDir); err == nil {
		if err := os.Rename(dstDir, dstDir+".rollback"); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(dstDir), 0o755); err != nil {
		return err
	}
	if err := os.Rename(srcDir, dstDir); err != nil {
		return err
	}
	_ = os.RemoveAll(dstDir + ".rollback")
	return nil
}

func restoreTable(tx *gorm.DB, raw json.RawMessage, dest any) error {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return err
	}
	if countJSONArray(raw) == 0 {
		return nil
	}
	return tx.Create(dest).Error
}

func disableForeignKeys(tx *gorm.DB) error {
	switch tx.Dialector.Name() {
	case "sqlite":
		return tx.Exec("PRAGMA foreign_keys = OFF").Error
	case "mysql":
		return tx.Exec("SET FOREIGN_KEY_CHECKS = 0").Error
	default:
		return nil
	}
}

func enableForeignKeys(tx *gorm.DB) error {
	switch tx.Dialector.Name() {
	case "sqlite":
		return tx.Exec("PRAGMA foreign_keys = ON").Error
	case "mysql":
		return tx.Exec("SET FOREIGN_KEY_CHECKS = 1").Error
	default:
		return nil
	}
}

func countJSONArray(raw []byte) int {
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return 0
	}
	return len(rows)
}
