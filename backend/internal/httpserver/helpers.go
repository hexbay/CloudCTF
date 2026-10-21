package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

// ================================================
// Model aliases & role constants
// ================================================

const (
	roleSuperAdmin = "superadmin"
	roleAdmin      = "admin"
	rolePlayer     = "player"
)

type userModel = models.User
type teamModel = models.Team
type challengeModel = models.Challenge
type theoryQuestionModel = models.TheoryQuestion
type examModel = models.Exam
type announcementModel = models.Announcement
type virtualizationHostModel = models.VirtualizationHost

// ================================================
// Request query / path helpers
// ================================================

func queryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func queryStr(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func pathUint(r *http.Request, key string) uint {
	val, err := strconv.ParseUint(r.PathValue(key), 10, 64)
	if err != nil {
		return 0
	}
	return uint(val)
}

// ================================================
// HTTP body / response helpers
// ================================================

func readJSONMap(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid multipart body"})
			return nil, false
		}
		return map[string]any{}, true
	}
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json body"})
		return nil, false
	}
	return payload, true
}

func resourceID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err == nil {
			return part
		}
	}
	return ""
}

func writeExport(w http.ResponseWriter, resource string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, resource))
	_, _ = w.Write([]byte("id,name,created_at\n"))
}

func stringifyID(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	default:
		return fmt.Sprint(value)
	}
}

// ================================================
// Value coercion helpers
// ================================================

func valueOrDefault(value any, fallback string) string {
	text := cleanString(value)
	if text == "" {
		return fallback
	}
	return text
}

func parseOptionalTime(value any) *time.Time {
	text := cleanString(value)
	if text == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.Parse(layout, text)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func uintValue(value any) uint {
	var parsed uint
	_, _ = fmt.Sscan(stringifyID(value), &parsed)
	return parsed
}

func cleanString(value any) string {
	if value == nil {
		return ""
	}
	text := fmt.Sprint(value)
	return strings.TrimSpace(text)
}

func boolValue(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		if typed == "true" {
			return true
		}
		if typed == "false" {
			return false
		}
	}
	return fallback
}

// ================================================
// JSON (un)marshalling helpers
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

func jsonString(value any) string {
	if value == nil {
		return "{}"
	}
	switch typed := value.(type) {
	case string:
		if typed == "" || typed == "<nil>" {
			return "{}"
		}
		return typed
	default:
		payload, err := json.Marshal(value)
		if err != nil {
			return "{}"
		}
		return string(payload)
	}
}

// ================================================
// Row serialization helpers
// ================================================

func userToRow(user userModel) map[string]any {
	return map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"full_name":    user.FullName,
		"is_active":    user.IsActive,
		"is_superuser": user.IsSuperuser,
		"tags":         user.TagList(),
		"role":         user.Role.Name,
		"role_id":      user.RoleID,
		"avatar":       user.Avatar,
		"teams":        []map[string]any{},
		"created_at":   user.CreatedAt,
		"updated_at":   user.UpdatedAt,
	}
}

func theoryToRow(item theoryQuestionModel) map[string]any {
	return map[string]any{
		"id":            item.ID,
		"question":      item.Question,
		"points":        item.Points,
		"category":      item.Category,
		"difficulty":    item.Difficulty,
		"question_type": item.QuestionType,
		"context":       item.ContextMap(),
		"created_by":    item.CreatedBy,
		"created_at":    item.CreatedAt,
		"updated_at":    item.UpdatedAt,
	}
}

// ================================================
// Security helpers
// ================================================

func verifyPassword(password, hashed string) bool {
	if strings.HasPrefix(hashed, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
	}
	legacy := sha256.Sum256([]byte(password))
	return hex.EncodeToString(legacy[:]) == hashed
}

func bearerToken(auth string) string {
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func challengeToRow(item challengeModel) map[string]any {
	return map[string]any{
		"id":                     item.ID,
		"name":                   item.Name,
		"description":            item.Description,
		"category":               item.Category,
		"points":                 item.Points,
		"difficulty":             item.Difficulty,
		"is_active":              item.IsActive,
		"is_public":              item.IsPublic,
		"usage":                  item.Usage,
		"is_dynamic":             item.IsDynamic,
		"container_image":        item.ContainerImage,
		"container_port":         item.ContainerPort,
		"container_timeout":      item.ContainerTimeout,
		"host_id":                item.HostID,
		"template_id":            item.TemplateID,
		"competition_id":         item.CompetitionID,
		"source_challenge_id":    item.SourceChallengeID,
		"level_one_solution":     item.LevelOneSolution,
		"has_level_one_solution": strings.TrimSpace(item.LevelOneSolution) != "",
		"hints":                  item.HintList(),
		"files":                  item.FileList(),
		"created_by":             item.CreatedBy,
		"created_at":             item.CreatedAt,
		"updated_at":             item.UpdatedAt,
	}
}

func intValue(value any, fallback int) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		var parsed int
		_, _ = fmt.Sscan(typed, &parsed)
		return parsed
	}
	return fallback
}

func optionalUint(value any) *uint {
	switch typed := value.(type) {
	case uint:
		if typed == 0 {
			return nil
		}
		v := typed
		return &v
	case uint64:
		if typed == 0 {
			return nil
		}
		v := uint(typed)
		return &v
	case int64:
		if typed == 0 {
			return nil
		}
		v := uint(typed)
		return &v
	case float64:
		v := uint(typed)
		if v == 0 {
			return nil
		}
		return &v
	case int:
		if typed == 0 {
			return nil
		}
		v := uint(typed)
		return &v
	case string:
		var parsed uint
		_, _ = fmt.Sscan(typed, &parsed)
		if parsed == 0 {
			return nil
		}
		return &parsed
	}
	return nil
}

func extractExamQuestionsFromContext(context map[string]any) []map[string]any {
	if rawQuestions, ok := context["questions"].([]any); ok && len(rawQuestions) > 0 {
		rows := make([]map[string]any, 0, len(rawQuestions))
		for _, raw := range rawQuestions {
			if row, ok := raw.(map[string]any); ok {
				rows = append(rows, normalizeExamQuestionRow(row))
			}
		}
		if len(rows) > 0 {
			return rows
		}
	}
	if rawQuestions, ok := context["questions"].([]map[string]any); ok && len(rawQuestions) > 0 {
		rows := make([]map[string]any, 0, len(rawQuestions))
		for _, row := range rawQuestions {
			rows = append(rows, normalizeExamQuestionRow(row))
		}
		return rows
	}
	if rawIDs, ok := context["question_ids"].([]any); ok && len(rawIDs) > 0 {
		rows := make([]map[string]any, 0, len(rawIDs))
		for _, rawID := range rawIDs {
			id := uintValue(rawID)
			if id == 0 {
				continue
			}
			rows = append(rows, map[string]any{"id": id})
		}
		return rows
	}
	if rawIDs, ok := context["question_ids"].([]uint); ok && len(rawIDs) > 0 {
		rows := make([]map[string]any, 0, len(rawIDs))
		for _, id := range rawIDs {
			rows = append(rows, map[string]any{"id": id})
		}
		return rows
	}
	return []map[string]any{}
}

func normalizeExamQuestionRow(row map[string]any) map[string]any {
	return map[string]any{
		"id":            uintValue(row["id"]),
		"question":      cleanString(row["question"]),
		"points":        intValue(row["points"], 0),
		"category":      cleanString(row["category"]),
		"difficulty":    intValue(row["difficulty"], 1),
		"question_type": valueOrDefault(row["question_type"], "text"),
		"context":       parseJSONObject(jsonString(row["context"])),
	}
}

func sanitizePlayerExamQuestionContext(context map[string]any) map[string]any {
	delete(context, "answer")
	if rawOptions, ok := context["options"].([]any); ok {
		options := make([]map[string]any, 0, len(rawOptions))
		for _, rawOption := range rawOptions {
			option, ok := rawOption.(map[string]any)
			if !ok {
				continue
			}
			options = append(options, map[string]any{"text": cleanString(option["text"])})
		}
		context["options"] = options
	}
	return context
}
