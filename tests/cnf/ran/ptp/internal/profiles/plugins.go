package profiles

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/ptp"
	ptpv1 "github.com/rh-ecosystem-edge/eco-goinfra/pkg/schemes/ptp/v1"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/iface"
)

// PinStateType enumerates the supported pin states.
type PinStateType int

const (
	// PinStateDisabled is the disabled state: 0.
	PinStateDisabled PinStateType = iota
	// PinStateRx is the RX state: 1.
	PinStateRx
	// PinStateTx is the TX state: 2.
	PinStateTx
)

// GetInterfacesWithPluginPins returns interface names that have at least one pin
// configured as the specified pin state in the profile's plugin pins. Returns nil if the
// profile has no plugin or no pins or an error occurs. Pins use "pin-state channel" syntax;
// pinState is the pin state to look for.
func GetInterfacesWithPluginPins(profile *ptpv1.PtpProfile,
	pluginType ptp.PluginType,
	pinState PinStateType) ([]iface.Name, error) {
	if profile == nil || profile.Plugins == nil {
		return nil, fmt.Errorf("profile is nil or has no plugins")
	}

	pluginJSON, ok := profile.Plugins[string(pluginType)]
	if !ok || pluginJSON == nil || len(pluginJSON.Raw) == 0 {
		return nil, fmt.Errorf("%s plugin not found in profile", pluginType)
	}

	var intelPlugin ptp.IntelPlugin
	if err := json.Unmarshal(pluginJSON.Raw, &intelPlugin); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s plugin: %w", pluginType, err)
	}

	if intelPlugin.Pins == nil {
		return nil, fmt.Errorf("%s plugin has no pins", pluginType)
	}

	var interfaceNames []iface.Name

	for ifaceName, connectorToValue := range intelPlugin.Pins {
		for _, value := range connectorToValue {
			first := strings.Fields(value)
			if len(first) >= 1 && first[0] == fmt.Sprintf("%d", pinState) {
				interfaceNames = append(interfaceNames, iface.Name(ifaceName))

				break
			}
		}
	}

	return interfaceNames, nil
}

// GetPluginTypesFromProfile returns the plugin types from the profile. Returns an error if the profile has no plugins.
func GetPluginTypesFromProfile(profile *ptpv1.PtpProfile) ([]ptp.PluginType, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}

	if profile.Plugins == nil {
		return nil, fmt.Errorf("profile has no plugins")
	}

	pluginTypes := make([]ptp.PluginType, 0, len(profile.Plugins))
	for pluginType := range profile.Plugins {
		pluginTypes = append(pluginTypes, ptp.PluginType(pluginType))
	}

	return pluginTypes, nil
}
