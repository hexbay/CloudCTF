package services

import (
	"strings"
)

func ParserContainerGroupTarget(target string) []string {
	if target == "" {
		return nil
	}
	return strings.Split(target, ",")
}
