package safety

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathPolicy confines manifest reads to canonical configured roots.
type PathPolicy struct {
	roots []string
}

func NewPathPolicy(roots []string) (*PathPolicy, error) {
	if len(roots) == 0 {
		return nil, errors.New("at least one manifest root is required")
	}
	canonical := make([]string, 0, len(roots))
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve manifest root: %w", err)
		}
		resolved, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, fmt.Errorf("resolve manifest root %q: %w", root, err)
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("manifest root %q is not a directory", root)
		}
		canonical = append(canonical, filepath.Clean(resolved))
	}
	if len(canonical) == 0 {
		return nil, errors.New("at least one valid manifest root is required")
	}
	return &PathPolicy{roots: canonical}, nil
}

func (p *PathPolicy) Roots() []string {
	return append([]string(nil), p.roots...)
}

func (p *PathPolicy) Resolve(candidate string) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", errors.New("manifest path is required")
	}
	paths := []string{candidate}
	if !filepath.IsAbs(candidate) {
		paths = paths[:0]
		for _, root := range p.roots {
			paths = append(paths, filepath.Join(root, candidate))
		}
	}
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		resolved, err := filepath.EvalSymlinks(abs)
		if err != nil {
			continue
		}
		info, err := os.Stat(resolved)
		if err != nil || info.IsDir() {
			continue
		}
		for _, root := range p.roots {
			if within(root, resolved) {
				return filepath.Clean(resolved), nil
			}
		}
	}
	return "", fmt.Errorf("manifest path %q escapes configured roots or is not a file", candidate)
}

func within(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
