package client

// Typed helpers for permission updates.
//
// These types serialize into the same JSON-ish objects Claude Code uses in
// permission_suggestions / updated_permissions. They do not change behavior;
// they just make it easier to build correct shapes.

type PermissionRule struct {
	ToolName    string
	RuleContent string
}

type PermissionUpdate struct {
	Type        string
	Rules       []PermissionRule
	Behavior    string
	Mode        string
	Directories []string
	Destination string
}

func (u PermissionUpdate) ToMap() map[string]any {
	out := map[string]any{
		"type": u.Type,
	}
	if u.Destination != "" {
		out["destination"] = u.Destination
	}
	if u.Behavior != "" {
		out["behavior"] = u.Behavior
	}
	if u.Mode != "" {
		out["mode"] = u.Mode
	}
	if len(u.Directories) > 0 {
		dirs := make([]any, 0, len(u.Directories))
		for _, dir := range u.Directories {
			dirs = append(dirs, dir)
		}
		out["directories"] = dirs
	}
	if len(u.Rules) > 0 {
		rules := make([]any, 0, len(u.Rules))
		for _, rule := range u.Rules {
			item := map[string]any{"toolName": rule.ToolName}
			if rule.RuleContent != "" {
				item["ruleContent"] = rule.RuleContent
			}
			rules = append(rules, item)
		}
		out["rules"] = rules
	}
	return out
}

func PermissionUpdateSetMode(mode string, destination string) PermissionUpdate {
	return PermissionUpdate{Type: "setMode", Mode: mode, Destination: destination}
}

func PermissionUpdateAddRules(behavior string, destination string, rules ...PermissionRule) PermissionUpdate {
	return PermissionUpdate{
		Type:        "addRules",
		Behavior:    behavior,
		Destination: destination,
		Rules:       append([]PermissionRule(nil), rules...),
	}
}

func PermissionUpdateReplaceRules(behavior string, destination string, rules ...PermissionRule) PermissionUpdate {
	return PermissionUpdate{
		Type:        "replaceRules",
		Behavior:    behavior,
		Destination: destination,
		Rules:       append([]PermissionRule(nil), rules...),
	}
}

func PermissionUpdateRemoveRules(behavior string, destination string, rules ...PermissionRule) PermissionUpdate {
	return PermissionUpdate{
		Type:        "removeRules",
		Behavior:    behavior,
		Destination: destination,
		Rules:       append([]PermissionRule(nil), rules...),
	}
}

func PermissionUpdateAddDirectories(destination string, dirs ...string) PermissionUpdate {
	return PermissionUpdate{
		Type:        "addDirectories",
		Destination: destination,
		Directories: append([]string(nil), dirs...),
	}
}

func PermissionUpdateRemoveDirectories(destination string, dirs ...string) PermissionUpdate {
	return PermissionUpdate{
		Type:        "removeDirectories",
		Destination: destination,
		Directories: append([]string(nil), dirs...),
	}
}
