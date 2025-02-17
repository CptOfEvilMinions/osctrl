package main

import (
	"fmt"
	"log"

	"github.com/jmpsec/osctrl/metrics"
	"github.com/jmpsec/osctrl/settings"
)

// Function to load the metrics settings
func loadingMetrics(mgr *settings.Settings) (*metrics.Metrics, error) {
	// Check if service settings for metrics is ready, initialize if so
	if !mgr.IsValue(settings.ServiceAdmin, settings.ServiceMetrics) {
		if err := mgr.NewBooleanValue(settings.ServiceAdmin, settings.ServiceMetrics, false); err != nil {
			return nil, fmt.Errorf("Failed to add %s to configuration: %v", settings.ServiceMetrics, err)
		}
	}
	if mgr.ServiceMetrics(settings.ServiceAdmin) {
		_mCfg, err := metrics.LoadConfiguration()
		if err != nil {
			if err := mgr.SetBoolean(false, settings.ServiceAdmin, settings.ServiceMetrics); err != nil {
				return nil, fmt.Errorf("Failed to disable metrics: %v", err)
			}
			return nil, fmt.Errorf("Failed to initialize metrics: %v", err)
		}
		_m, err := metrics.CreateMetrics(_mCfg.Protocol, _mCfg.Host, _mCfg.Port, serviceName)
		if err != nil {
			if err := mgr.SetBoolean(false, settings.ServiceAdmin, settings.ServiceMetrics); err != nil {
				return nil, fmt.Errorf("Failed to disable metrics: %v", err)
			}
			return nil, fmt.Errorf("Failed to initialize metrics: %v", err)
		}
		return _m, nil
	}
	return nil, nil
}

// Function to load all settings for the service
func loadingSettings(mgr *settings.Settings) error {
	// Check if service settings for debug service is ready
	if mgr.DebugService(settings.ServiceAdmin) {
		log.Println("DebugService: Initializing settings")
	}
	// Check if service settings for debug service is ready
	if !mgr.IsValue(settings.ServiceAdmin, settings.DebugService) {
		if err := mgr.NewBooleanValue(settings.ServiceAdmin, settings.DebugService, false); err != nil {
			return fmt.Errorf("Failed to add %s to settings: %v", settings.DebugService, err)
		}
	}
	// Check if service settings for debug HTTP is ready
	if !mgr.IsValue(settings.ServiceAdmin, settings.DebugHTTP) {
		if err := mgr.NewBooleanValue(settings.ServiceAdmin, settings.DebugHTTP, false); err != nil {
			return fmt.Errorf("Failed to add %s to settings: %v", settings.DebugHTTP, err)
		}
	}
	// Check if service settings for default environment is ready
	if !mgr.IsValue(settings.ServiceAdmin, settings.DefaultEnv) {
		if err := mgr.NewStringValue(settings.ServiceAdmin, settings.DefaultEnv, "dev"); err != nil {
			return fmt.Errorf("Failed to add %s to settings: %v", settings.DefaultEnv, err)
		}
	}
	// Check if service settings for sessions cleanup is ready
	if !mgr.IsValue(settings.ServiceAdmin, settings.CleanupSessions) {
		if err := mgr.NewIntegerValue(settings.ServiceAdmin, settings.CleanupSessions, int64(defaultRefresh)); err != nil {
			return fmt.Errorf("Failed to add %s to configuration: %v", settings.CleanupSessions, err)
		}
	}
	// Check if service settings for node inactive hours is ready
	if !mgr.IsValue(settings.ServiceAdmin, settings.InactiveHours) {
		if err := mgr.NewIntegerValue(settings.ServiceAdmin, settings.InactiveHours, int64(defaultInactive)); err != nil {
			return fmt.Errorf("Failed to add %s to configuration: %v", settings.InactiveHours, err)
		}
	}
	// Check if service settings for display dashboard is ready
	if !mgr.IsValue(settings.ServiceAdmin, settings.NodeDashboard) {
		if err := mgr.NewBooleanValue(settings.ServiceAdmin, settings.NodeDashboard, false); err != nil {
			return fmt.Errorf("Failed to add %s to settings: %v", settings.NodeDashboard, err)
		}
	}
	// Write JSON config to settings
	if err := mgr.SetAdminJSON(adminConfig); err != nil {
		return fmt.Errorf("Failed to add JSON values to configuration: %v", err)
	}
	return nil
}
