package httpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

func (srv *Server) loadSystemSettings() map[string]any {
	service := srv.getSystemSettingService()
	if service == nil {
		return defaultSettings()
	}
	return service.Load(defaultSettings())
}

func (srv *Server) saveSystemSettings(payload map[string]any) error {
	service := srv.getSystemSettingService()
	if service == nil {
		return fmt.Errorf("store unavailable")
	}
	return service.Save(payload, defaultSettings())
}

func (srv *Server) systemSettingBool(key string, fallback bool) bool {
	return boolValue(srv.loadSystemSettings()[key], fallback)
}

func (srv *Server) systemSettingInt(key string, fallback int) int {
	return intValue(srv.loadSystemSettings()[key], fallback)
}

func (srv *Server) publicSystemSettings() map[string]any {
	settings := srv.loadSystemSettings()
	return map[string]any{
		"site_name":            settings["site_name"],
		"site_title":           settings["site_title"],
		"site_logo":            settings["site_logo"],
		"site_description":     settings["site_description"],
		"allow_registration":   settings["allow_registration"],
		"enable_notifications": settings["enable_notifications"],
		"enable_writeup":       settings["enable_writeup"],
		"show_github_link":     boolValue(settings["show_github_link"], true),
		"max_team_size":        settings["max_team_size"],
	}
}

func (srv *Server) systemInfoSnapshot() map[string]any {
	stats := srv.collectSystemStats()
	return map[string]any{
		"cpu_percent":    stats.CPUPercent,
		"memory_percent": stats.MemoryPercent,
		"disk_percent":   stats.DiskPercent,
		"db_size":        srv.currentDatabaseSize(),
		"uptime":         srv.currentUptime(),
		"hostname":       stats.Hostname,
		"os":             stats.OS,
		"arch":           stats.Arch,
		"last_updated":   time.Now().Format(time.RFC3339),
	}
}

type systemStats struct {
	CPUPercent    float64
	MemoryPercent float64
	DiskPercent   float64
	Hostname      string
	OS            string
	Arch          string
}

func (srv *Server) collectSystemStats() systemStats {
	stats := systemStats{
		Hostname: "unknown",
		OS:       "unknown",
		Arch:     "unknown",
	}
	if values, err := cpu.Percent(0, false); err == nil && len(values) > 0 {
		stats.CPUPercent = roundPercent(values[0])
	}
	if values, err := mem.VirtualMemory(); err == nil && values != nil {
		stats.MemoryPercent = roundPercent(values.UsedPercent)
	}
	if usage, err := disk.Usage(srv.runtimeDiskPath()); err == nil && usage != nil {
		stats.DiskPercent = roundPercent(usage.UsedPercent)
	}
	if info, err := host.Info(); err == nil && info != nil {
		if trimmed := strings.TrimSpace(info.Hostname); trimmed != "" {
			stats.Hostname = trimmed
		}
		if trimmed := strings.TrimSpace(info.Platform); trimmed != "" {
			stats.OS = trimmed
		}
		if trimmed := strings.TrimSpace(info.KernelArch); trimmed != "" {
			stats.Arch = trimmed
		}
	}
	return stats
}

func (srv *Server) currentDatabaseSize() string {
	path, ok := sqlitePathFromURI(srv.currentConfig().DatabaseURI)
	if !ok {
		return "external"
	}
	info, err := os.Stat(path)
	if err != nil {
		return "0B"
	}
	return humanBytes(info.Size())
}

func sqlitePathFromURI(uri string) (string, bool) {
	const prefix = "sqlite://"
	if !strings.HasPrefix(uri, prefix) {
		return "", false
	}
	return strings.TrimPrefix(uri, prefix), true
}

func (srv *Server) currentUptime() string {
	booted := srv.currentBootedAt()
	if booted.IsZero() {
		return "unknown"
	}
	return time.Since(booted).Round(time.Second).String()
}

func (srv *Server) runtimeDiskPath() string {
	cfg := srv.currentConfig()
	if path, ok := sqlitePathFromURI(cfg.DatabaseURI); ok {
		if abs, err := filepath.Abs(path); err == nil {
			return filepath.Dir(abs)
		}
		return filepath.Dir(path)
	}
	if strings.TrimSpace(cfg.BackupDir) != "" {
		return cfg.BackupDir
	}
	return "."
}

func roundPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return float64(int(value*10+0.5)) / 10
}

func humanBytes(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d%s", size, units[unit])
	}
	if value >= 10 {
		return fmt.Sprintf("%.0f%s", value, units[unit])
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}
