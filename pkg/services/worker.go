package services

import (
	"encoding/json"
	"fmt"
)

func (s *service) PingWorker() (string, error) {
	if s.bridge == nil {
		return "", fmt.Errorf("bridge is not initialized")
	}

	result, err := s.bridge.Call("ping", nil)
	if err != nil {
		return "", err
	}

	var response string
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("failed to parse worker response: %w", err)
	}

	return response, nil
}
