package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type commit struct {
	SHA     string
	Subject string
	Body    string
}

func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output() // NOSONAR
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		err = errors.New(strings.TrimSpace(string(exitErr.Stderr)))
	}
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func listCommits(revisionRange string) ([]commit, error) {
	out, err := git("log", "--reverse", "--format=%H%x1f%s%x1f%b%x1e", "--end-of-options", revisionRange)
	if err != nil {
		return nil, err
	}
	var commits []commit
	for record := range strings.SplitSeq(out, "\x1e") {
		if fields := strings.SplitN(strings.TrimSpace(record), "\x1f", 3); len(fields) == 3 {
			commits = append(commits, commit{SHA: fields[0], Subject: fields[1], Body: fields[2]})
		}
	}
	return commits, nil
}

var gitHubRemotePattern = regexp.MustCompile(`github\.com[:/]+([^/]+/[^/]+?)(\.git)?/?$`)

func gitHubRepository() (string, error) {
	if repository := os.Getenv("GITHUB_REPOSITORY"); repository != "" {
		return repository, nil
	}
	remote, err := git("remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	match := gitHubRemotePattern.FindStringSubmatch(remote)
	if match == nil {
		return "", fmt.Errorf("remote %q is no GitHub repository", remote)
	}
	return match[1], nil
}
