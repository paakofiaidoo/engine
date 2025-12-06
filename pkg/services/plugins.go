package services

import (
	"encoding/json"
	"fmt"
)

func (s *service) InstallPlugin(projectID, pluginName, version string) error {
	if s.bridge == nil {
		return fmt.Errorf("bridge is not initialized")
	}

	params := map[string]string{
		"project_id":  projectID,
		"plugin_name": pluginName,
		"version":     version,
	}

	result, err := s.bridge.Call("install_plugin", params)
	if err != nil {
		return err
	}

	// Parse result if needed, for now just check for error
	var response map[string]interface{}
	if err := json.Unmarshal(result, &response); err != nil {
		return fmt.Errorf("failed to parse worker response: %w", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		return fmt.Errorf("worker failed to install plugin: %v", response["message"])
	}

	return nil
}

func (s *service) ConfigureCMS(projectID, cmsType, configJSON string) error {
	if s.bridge == nil {
		return fmt.Errorf("bridge is not initialized")
	}

	params := map[string]string{
		"project_id":  projectID,
		"cms_type":    cmsType,
		"config_json": configJSON,
	}

	result, err := s.bridge.Call("configure_cms", params)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(result, &response); err != nil {
		return fmt.Errorf("failed to parse worker response: %w", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		return fmt.Errorf("worker failed to configure cms: %v", response["message"])
	}

	return nil
}
