package protocol

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ResolveInsideRoot(root, requested string) (string, error) {
	return resolveInsideRoot(root, requested, false)
}

func ResolveInsideRootNoFollow(root, requested string) (string, error) {
	root = strings.TrimSpace(root)
	requested = strings.TrimSpace(requested)
	if root == "" {
		return "", fmt.Errorf("root is required")
	}
	if requested == "" {
		requested = "."
	}
	if filepath.IsAbs(requested) {
		return "", NewError(ErrorPathOutsideRoot, "absolute paths are not accepted by the ccgo file layer")
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	cleanRoot, err = filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(cleanRoot, requested))
	if err != nil {
		return "", err
	}
	if err := ensureInsideRoot(cleanRoot, candidate); err != nil {
		return "", err
	}
	parent := filepath.Dir(candidate)
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	if err := ensureInsideRoot(cleanRoot, resolvedParent); err != nil {
		return "", err
	}
	return filepath.Join(resolvedParent, filepath.Base(candidate)), nil
}

func ResolveLocalPathInsideRoot(root, requested string) (string, error) {
	return resolveInsideRoot(root, requested, true)
}

func resolveInsideRoot(root, requested string, allowAbsolute bool) (string, error) {
	root = strings.TrimSpace(root)
	requested = strings.TrimSpace(requested)
	if root == "" {
		return "", fmt.Errorf("root is required")
	}
	if requested == "" {
		requested = "."
	}
	if filepath.IsAbs(requested) && !allowAbsolute {
		return "", NewError(ErrorPathOutsideRoot, "absolute paths are not accepted by the ccgo file layer")
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	cleanRoot, err = filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		return "", err
	}
	candidate := requested
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(cleanRoot, candidate)
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	return resolveCandidateInsideRoot(cleanRoot, candidate)
}

func resolveCandidateInsideRoot(cleanRoot, candidate string) (string, error) {
	candidate = filepath.Clean(candidate)
	if runtime.GOOS == "windows" && filepath.VolumeName(candidate) != filepath.VolumeName(cleanRoot) {
		return "", NewError(ErrorPathOutsideRoot, "path escapes the ccgo workspace volume")
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err == nil {
		if err := ensureInsideRoot(cleanRoot, resolved); err != nil {
			return "", err
		}
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	existing := candidate
	missingParts := make([]string, 0, 2)
	for {
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", err
		}
		missingParts = append([]string{filepath.Base(existing)}, missingParts...)
		existing = parent

		resolvedParent, parentErr := filepath.EvalSymlinks(existing)
		if parentErr == nil {
			if err := ensureInsideRoot(cleanRoot, resolvedParent); err != nil {
				return "", err
			}
			resolved = filepath.Join(append([]string{resolvedParent}, missingParts...)...)
			if err := ensureInsideRoot(cleanRoot, resolved); err != nil {
				return "", err
			}
			return resolved, nil
		}
		if !os.IsNotExist(parentErr) {
			return "", parentErr
		}
	}
}

func ensureInsideRoot(cleanRoot, candidate string) error {
	rel, err := filepath.Rel(cleanRoot, candidate)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return NewError(ErrorPathOutsideRoot, "path escapes the ccgo workspace root")
	}
	if runtime.GOOS == "windows" && filepath.VolumeName(candidate) != filepath.VolumeName(cleanRoot) {
		return NewError(ErrorPathOutsideRoot, "path escapes the ccgo workspace volume")
	}
	return nil
}

func RelativePath(root, absolute string) (string, error) {
	rel, err := filepath.Rel(root, absolute)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return ".", nil
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", NewError(ErrorPathOutsideRoot, "path escapes the ccgo workspace root")
	}
	return rel, nil
}
