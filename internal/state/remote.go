package state

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/hra42/deployer/internal/sshx"
)

// mergeScript is executed remotely under flock -x. It reads an existing JSON
// state file (treating missing or invalid as empty), merges sys.argv[2]
// (a JSON object patch), and atomically writes the result back.
const mergeScript = `import json,os,sys
path,patch_json,key=sys.argv[1],sys.argv[2],sys.argv[3]
patch=json.loads(patch_json)
try:
    with open(path) as f: state=json.load(f) or {}
except FileNotFoundError: state={}
except json.JSONDecodeError: state={}
state.update(patch)
tmp=path+'.tmp'
with open(tmp,'w') as f: json.dump(state,f,indent=2,sort_keys=True)
os.replace(tmp,path)
`

func paths(clonePath string) (statePath, lockPath string) {
	return path.Join(clonePath, StateFile), path.Join(clonePath, LockFile)
}

// Load reads the remote state file. A missing or empty file returns an empty
// State without error.
func Load(ctx context.Context, c *sshx.Client, clonePath string) (State, error) {
	out, err := c.Output(ctx, buildLoadCmd(clonePath))
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}
	trimmed := strings.TrimSpace(out)
	if trimmed == "" || trimmed == "{}" {
		return State{}, nil
	}
	var s State
	if err := json.Unmarshal([]byte(trimmed), &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if s == nil {
		s = State{}
	}
	return s, nil
}

// Upsert inserts or replaces the entry for domain in the remote state file.
// The read-modify-write happens inside a single flock -x'd python3 invocation
// so concurrent deploys cannot clobber each other.
func Upsert(ctx context.Context, c *sshx.Client, clonePath, domain string, svc Service) error {
	cmd, err := buildUpsertCmd(clonePath, domain, svc)
	if err != nil {
		return err
	}
	if err := c.Run(ctx, cmd); err != nil {
		return fmt.Errorf("upsert state: %w", err)
	}
	return nil
}

func buildLoadCmd(clonePath string) string {
	statePath, lockPath := paths(clonePath)
	inner := "cat " + shellQuote(statePath) + " 2>/dev/null || echo '{}'"
	return fmt.Sprintf(
		"mkdir -p %s && touch %s && flock -s %s sh -c %s",
		shellQuote(clonePath),
		shellQuote(lockPath),
		shellQuote(lockPath),
		shellQuote(inner),
	)
}

func buildUpsertCmd(clonePath, domain string, svc Service) (string, error) {
	if domain == "" {
		return "", fmt.Errorf("domain is empty")
	}
	patch := map[string]Service{domain: svc}
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return "", fmt.Errorf("marshal patch: %w", err)
	}
	statePath, lockPath := paths(clonePath)
	return fmt.Sprintf(
		"mkdir -p %s && touch %s && flock -x %s python3 -c %s %s %s %s",
		shellQuote(clonePath),
		shellQuote(lockPath),
		shellQuote(lockPath),
		shellQuote(mergeScript),
		shellQuote(statePath),
		shellQuote(string(patchJSON)),
		shellQuote(domain),
	), nil
}
