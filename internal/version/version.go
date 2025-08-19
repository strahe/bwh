package version

import (
	"fmt"
	"runtime/debug"
)

var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "unknown"
)

func GetVersion() string {
	if Version != "dev" && Version != "unknown" {
		return Version
	}
	
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) >= 7 {
					return setting.Value[:7]
				}
				return setting.Value
			}
		}
		
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
	}
	
	return Version
}

func GetBuildTime() string {
	if BuildTime != "unknown" {
		return BuildTime
	}
	
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" {
				return setting.Value
			}
		}
	}
	
	return BuildTime
}

func GetCommitHash() string {
	if CommitHash != "unknown" {
		return CommitHash
	}
	
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	
	return CommitHash
}

func GetUserAgent() string {
	version := GetVersion()
	if version == "dev" || version == "unknown" {
		return "BWH-CLI/dev"
	}
	return fmt.Sprintf("BWH-CLI/%s", version)
}

func GetFullVersionInfo() string {
	return fmt.Sprintf("BWH CLI %s (built %s, commit %s)", GetVersion(), GetBuildTime(), GetCommitHash())
}