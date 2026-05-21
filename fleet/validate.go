package fleet

import (
	"fmt"
	"sort"
	"strings"
)

const ManifestSchemaVersion = "vegm.fleet-manifest/v1"

func ValidateManifest(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("manifest is nil")
	}
	var errs []string
	if m.SchemaVersion != ManifestSchemaVersion {
		errs = append(errs, fmt.Sprintf("schema_version must be %q", ManifestSchemaVersion))
	}
	if strings.TrimSpace(m.FleetName) == "" {
		errs = append(errs, "fleet_name is required")
	}
	if len(m.Profiles) == 0 {
		errs = append(errs, "profiles must not be empty")
	}
	if len(m.Groups) == 0 {
		errs = append(errs, "groups must not be empty")
	}
	if len(m.Instances) == 0 {
		errs = append(errs, "instances must not be empty")
	}
	for name, p := range m.Profiles {
		if strings.TrimSpace(name) == "" {
			errs = append(errs, "profiles contains empty key")
			continue
		}
		if strings.TrimSpace(p.PackFile) == "" {
			errs = append(errs, fmt.Sprintf("profiles.%s.pack_file is required", name))
		}
		if len(p.LogicalCommands) == 0 {
			errs = append(errs, fmt.Sprintf("profiles.%s.logical_commands must not be empty", name))
		}
	}
	for name, g := range m.Groups {
		if strings.TrimSpace(name) == "" {
			errs = append(errs, "groups contains empty key")
			continue
		}
		if strings.TrimSpace(g.Profile) == "" {
			errs = append(errs, fmt.Sprintf("groups.%s.profile is required", name))
			continue
		}
		if _, ok := m.Profiles[g.Profile]; !ok {
			errs = append(errs, fmt.Sprintf("groups.%s.profile %q not found in profiles", name, g.Profile))
		}
	}
	seenInstanceIDs := map[string]bool{}
	seenEGMIDs := map[string]bool{}
	seenWirePorts := map[int]string{}
	seenControlPorts := map[int]string{}
	for i, inst := range m.Instances {
		if strings.TrimSpace(inst.InstanceID) == "" {
			errs = append(errs, fmt.Sprintf("instances[%d].instance_id is required", i))
		} else if seenInstanceIDs[inst.InstanceID] {
			errs = append(errs, fmt.Sprintf("instances[%d].instance_id %q is duplicated", i, inst.InstanceID))
		} else {
			seenInstanceIDs[inst.InstanceID] = true
		}
		if strings.TrimSpace(inst.EGMID) == "" {
			errs = append(errs, fmt.Sprintf("instances[%d].egm_id is required", i))
		} else if seenEGMIDs[inst.EGMID] {
			errs = append(errs, fmt.Sprintf("instances[%d].egm_id %q is duplicated", i, inst.EGMID))
		} else {
			seenEGMIDs[inst.EGMID] = true
		}
		if strings.TrimSpace(inst.Group) == "" {
			errs = append(errs, fmt.Sprintf("instances[%d].group is required", i))
		} else if _, ok := m.Groups[inst.Group]; !ok {
			errs = append(errs, fmt.Sprintf("instances[%d].group %q not found in groups", i, inst.Group))
		}
		if inst.WirePort < 0 {
			errs = append(errs, fmt.Sprintf("instances[%d].wire_port must be >= 0", i))
		}
		if inst.ControlPort < 0 {
			errs = append(errs, fmt.Sprintf("instances[%d].control_port must be >= 0", i))
		}
		if inst.WirePort > 0 {
			if prior, ok := seenWirePorts[inst.WirePort]; ok {
				errs = append(errs, fmt.Sprintf("instances[%d].wire_port %d duplicates instance %q", i, inst.WirePort, prior))
			} else {
				seenWirePorts[inst.WirePort] = inst.InstanceID
			}
		}
		if inst.ControlPort > 0 {
			if prior, ok := seenControlPorts[inst.ControlPort]; ok {
				errs = append(errs, fmt.Sprintf("instances[%d].control_port %d duplicates instance %q", i, inst.ControlPort, prior))
			} else {
				seenControlPorts[inst.ControlPort] = inst.InstanceID
			}
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("fleet manifest validation failed:\n- %s", strings.Join(errs, "\n- "))
	}
	return nil
}
