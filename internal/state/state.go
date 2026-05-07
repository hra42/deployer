// Package state manages the remote per-host JSON state file that records
// which services have been deployed. Access is mediated over SSH and
// serialized with flock on the host.
package state

import (
	"strings"
	"time"
)

const (
	StateFile = ".deployer-state.json"
	LockFile  = ".deployer-state.lock"
)

type Service struct {
	Repo       string    `json:"repo"`
	Slug       string    `json:"slug"`
	ClonePath  string    `json:"clone_path"`
	DeployedAt time.Time `json:"deployed_at"`
}

type State map[string]Service

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
