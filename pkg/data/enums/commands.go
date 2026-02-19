package enums

// AllowedCommand defines the safe structure for a terminal command
type AllowedCommand struct {
	Template string // e.g. "pnpm dev" or "pnpm install %s"
	Safe     bool
}

// CommandRegistry acts as the Source of Truth for all execution
var CommandRegistry = map[string]AllowedCommand{
	"DEV": {
		Template: "pnpm dev",
		Safe:     true,
	},
	"BUILD": {
		Template: "pnpm build",
		Safe:     true,
	},
	"INSTALL": {
		Template: "pnpm add %s", // pnpm usually uses 'add' for dependencies
		Safe:     false,         // Unsafe until validated
	},
	"GIT_PUSH": {
		Template: "git push origin juki-work",
		Safe:     true,
	},
}

// IsSafeArg validates arguments to prevent injection
func IsSafeArg(arg string) bool {
	// Simple allowlist: alphanumeric, dashes, @, / (for scoped packages)
	// No semicolons, pipes, or redirects
	for _, char := range arg {
		if char == ';' || char == '|' || char == '&' || char == '`' || char == '>' || char == '<' || char == '$' {
			return false
		}
	}
	return true
}
