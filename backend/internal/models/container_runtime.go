package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type ContainerTarget struct {
	// Port 为宿主机暴露端口。
	Port          any    `json:"port"`
	// Protocol 为访问协议，例如 http、tcp。
	Protocol      string `json:"protocol"`
	// ContainerPort 为容器内部端口。
	ContainerPort any    `json:"container_port"`
	// URL 为选手或管理员可直接访问的目标地址。
	URL           string `json:"url"`
}

// ContainerPorts 表示容器端口信息，兼容历史 map 与 array 两种存储格式。
type ContainerPorts struct {
	value any
}

func (p ContainerPorts) Value() any {
	if p.value == nil {
		return map[string]any{}
	}
	return p.value
}

func (p ContainerPorts) MarshalJSON() ([]byte, error) {
	if p.value == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(p.value)
}

func (p *ContainerPorts) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("container ports target is nil")
	}
	trimmed := string(data)
	if trimmed == "" || trimmed == "null" {
		p.value = map[string]any{}
		return nil
	}

	asMap := map[string]any{}
	if err := json.Unmarshal(data, &asMap); err == nil {
		p.value = asMap
		return nil
	}

	asArray := []any{}
	if err := json.Unmarshal(data, &asArray); err == nil {
		p.value = asArray
		return nil
	}

	return fmt.Errorf("unsupported container ports json: %s", trimmed)
}

// ContainerRuntime 表示容器组中单个运行中服务的运行时信息。
type ContainerRuntime struct {
	// ID 为运行时记录标识，通常与容器 ID 一致。
	ID          string            `json:"id"`
	// ContainerID 为 Docker 容器 ID。
	ContainerID string            `json:"container_id"`
	// Name 为容器名称。
	Name        string            `json:"name"`
	// Image 为容器镜像名。
	Image       string            `json:"image"`
	// Status 为容器当前状态。
	Status      string            `json:"status"`
	// State 为底层运行时状态字段。
	State       string            `json:"state"`
	// ServiceName 为 compose/模板中的服务名。
	ServiceName string            `json:"service_name"`
	// Ports 为端口信息，兼容宿主机映射表与历史数组格式。
	Ports       ContainerPorts    `json:"ports"`
	// Targets 为整理后的可访问目标列表。
	Targets     []ContainerTarget `json:"targets"`
	// CreatedAt 为容器创建时间。
	CreatedAt   any               `json:"created_at"`
	// UpdatedAt 为容器最近更新时间。
	UpdatedAt   any               `json:"updated_at"`
	// ExpiresAt 为容器或环境过期时间。
	ExpiresAt   any               `json:"expires_at"`
	// Flag 为动态环境可能附带的 flag 信息。
	Flag        string            `json:"flag"`
}

// ContainerRuntimeList 表示一组容器运行时信息，并负责 JSON 持久化。
type ContainerRuntimeList []ContainerRuntime

func (ContainerRuntimeList) GormDataType() string {
	return "json"
}

func (ContainerRuntimeList) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db != nil && db.Dialector != nil {
		switch db.Dialector.Name() {
		case "mysql":
			return "JSON"
		}
	}
	return "TEXT"
}

func (l ContainerRuntimeList) Value() (driver.Value, error) {
	if len(l) == 0 {
		return "[]", nil
	}
	payload, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(payload), nil
}

func (l *ContainerRuntimeList) Scan(value any) error {
	if l == nil {
		return fmt.Errorf("container runtime list scan target is nil")
	}
	switch typed := value.(type) {
	case nil:
		*l = ContainerRuntimeList{}
		return nil
	case []byte:
		return l.unmarshalJSONBytes(typed)
	case string:
		return l.unmarshalJSONBytes([]byte(typed))
	default:
		return fmt.Errorf("unsupported container runtime list type %T", value)
	}
}

func (l *ContainerRuntimeList) unmarshalJSONBytes(raw []byte) error {
	if len(raw) == 0 || string(raw) == "null" {
		*l = ContainerRuntimeList{}
		return nil
	}
	items := ContainerRuntimeList{}
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	*l = items
	return nil
}
