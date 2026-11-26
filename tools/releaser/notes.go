package main

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	sectionBreaking = "Breaking changes"
	sectionFeatures = "Features"
	sectionFixes    = "Fixes"
)

type entry struct {
	Commit    commit
	Section   string
	Text      string
	Reference string
	Author    string
	Issues    []int
}

var titlePattern = regexp.MustCompile(`^([A-Za-z]+)(\([^)]+\))?(!)?: (.+)$`)

func parseTitle(subject string) (section, description string) {
	match := titlePattern.FindStringSubmatch(subject)
	switch {
	case match == nil:
		return "", ""
	case match[3] == "!":
		return sectionBreaking, match[4]
	case match[1] == "feat":
		return sectionFeatures, match[4]
	case match[1] == "fix":
		return sectionFixes, match[4]
	}
	return "", ""
}

const aiAuthorshipMarkerPrefix = "🤖"

func leadingQuotes(text string) []string {
	var blocks []string
	var current []string
	flush := func() {
		if block := strings.TrimSpace(strings.Join(current, "\n")); block != "" && !strings.HasPrefix(block, aiAuthorshipMarkerPrefix) {
			blocks = append(blocks, block)
		}
		current = nil
	}
	for line := range strings.Lines(strings.ReplaceAll(text, "\r\n", "\n")) {
		line = strings.TrimSpace(line)
		quoted, isQuote := strings.CutPrefix(line, ">")
		switch {
		case isQuote:
			current = append(current, strings.TrimPrefix(quoted, " "))
		case line == "":
			flush()
		default:
			flush()
			return blocks
		}
	}
	flush()
	return blocks
}

func collectEntries(commits []commit) []entry {
	var entries []entry
	var shas []string
	for _, c := range commits {
		if section, description := parseTitle(c.Subject); section != "" {
			entries = append(entries, entry{Commit: c, Section: section, Text: description, Reference: c.SHA[:7]})
			shas = append(shas, c.SHA)
		}
	}
	if len(entries) == 0 {
		return nil
	}

	nodes, err := lookupCommits(shas)
	if err != nil {
		warn("Cannot read the pull requests from GitHub: %v. Every entry uses the commit text and names no author.", err)
	}
	pulls := map[int]*pullRequest{}
	entryCommits := map[int][]string{}
	for _, sha := range shas {
		if pr := nodes[sha].mergedPullRequest(); pr != nil {
			if _, seen := pulls[pr.Number]; !seen {
				pulls[pr.Number] = pr
			}
			entryCommits[pr.Number] = append(entryCommits[pr.Number], sha)
		}
	}
	blocks := map[int][]string{}
	issues := map[string][]int{}
	for _, number := range slices.Sorted(maps.Keys(pulls)) {
		maps.Copy(issues, assignIssues(pulls[number].ClosingIssuesReferences.Nodes, entryCommits[number]))
		quotes, want := leadingQuotes(pulls[number].Body), len(entryCommits[number])
		if len(quotes) != want {
			warn("The description of #%d starts with %d blockquote(s), and the pull request has %d feat, fix or breaking commit(s) in this range. Its entries use the commit text.", number, len(quotes), want)
			continue
		}
		blocks[number] = quotes
	}

	for i := range entries {
		e := &entries[i]
		node := nodes[e.Commit.SHA]
		e.Author, e.Issues = node.Author.User.Login, issues[e.Commit.SHA]
		text := strings.Join(leadingQuotes(e.Commit.Body), "\n\n")
		if pr := node.mergedPullRequest(); pr != nil {
			e.Reference = "#" + strconv.Itoa(pr.Number)
			if remaining := blocks[pr.Number]; len(remaining) > 0 {
				text, blocks[pr.Number] = remaining[0], remaining[1:]
			}
		}
		if text != "" {
			e.Text = text
		}
	}
	return entries
}

func assignIssues(closed []issue, entryCommits []string) map[string][]int {
	assigned := map[string][]int{}
	var leftOver []int
	for _, closing := range closed {
		if closer := closing.closerCommit(); slices.Contains(entryCommits, closer) {
			assigned[closer] = append(assigned[closer], closing.Number)
		} else {
			leftOver = append(leftOver, closing.Number)
		}
	}
	var withoutIssue []string
	for _, sha := range entryCommits {
		if len(assigned[sha]) == 0 {
			withoutIssue = append(withoutIssue, sha)
		}
	}
	switch {
	case len(leftOver) == 0:
	case len(entryCommits) == 1:
		assigned[entryCommits[0]] = append(assigned[entryCommits[0]], leftOver...)
	case len(withoutIssue) == 1:
		assigned[withoutIssue[0]] = leftOver
	}
	for _, numbers := range assigned {
		slices.Sort(numbers)
	}
	return assigned
}

func render(minMeshStackVersion string, entries []entry) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Requires meshStack %s or later.\n", minMeshStackVersion)
	for _, section := range []string{sectionBreaking, sectionFeatures, sectionFixes} {
		heading := "\n## " + section + "\n"
		for _, e := range entries {
			if e.Section == section {
				out.WriteString(heading + renderItem(e))
				heading = ""
			}
		}
	}
	return out.String()
}

const trailingWhitespace = " \t\r"

var (
	blockStartPattern = regexp.MustCompile("^\\s*([-*+] |\\d+[.)] |#{1,6} |```|~~~|\\||>|<)")
	fencePattern      = regexp.MustCompile("^\\s*(```|~~~)")
	standalonePattern = regexp.MustCompile(`^\s*(#{1,6} |\|)`)
)

func unwrap(text string) string {
	var out []string
	inFence, joinable := false, false
	for line := range strings.SplitSeq(text, "\n") {
		switch {
		case fencePattern.MatchString(line):
			inFence = !inFence
			out = append(out, line)
			joinable = false
		case inFence:
			out = append(out, line)
		case strings.TrimSpace(line) == "":
			out = append(out, "")
			joinable = false
		case blockStartPattern.MatchString(line) || !joinable:
			out = append(out, strings.TrimRight(line, trailingWhitespace))
			joinable = !standalonePattern.MatchString(line)
		default:
			out[len(out)-1] += " " + strings.TrimSpace(line)
		}
	}
	return strings.Join(out, "\n")
}

func (e entry) suffix() string {
	var out strings.Builder
	out.WriteString(" (" + e.Reference)
	if e.Author != "" {
		out.WriteString(" by @" + e.Author)
	}
	separator := ", closes"
	for _, issue := range e.Issues {
		fmt.Fprintf(&out, "%s #%d", separator, issue)
		separator = ","
	}
	out.WriteString(")")
	return out.String()
}

func renderItem(e entry) string {
	var out strings.Builder
	for i, line := range strings.Split(unwrap(strings.TrimSpace(e.Text)), "\n") {
		line = strings.TrimRight(line, trailingWhitespace)
		switch {
		case i == 0:
			out.WriteString("- " + line + e.suffix())
		case line != "":
			out.WriteString("  " + line)
		}
		out.WriteString("\n")
	}
	return out.String()
}

var versionTagPattern = regexp.MustCompile(`^(v\d+)\.(\d+)\.(\d+)$`)

func nextVersionHint(previousTag string, commits []commit) string {
	match := versionTagPattern.FindStringSubmatch(previousTag)
	if match == nil {
		return ""
	}
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	for _, c := range commits {
		if section, _ := parseTitle(c.Subject); section == sectionBreaking {
			return fmt.Sprintf("Next version: %s.%d.0 (breaking change since %s: %s %s)", match[1], minor+1, previousTag, c.SHA[:7], c.Subject)
		}
	}
	return fmt.Sprintf("Next version: %s.%d.%d (no breaking change since %s)", match[1], minor, patch+1, previousTag)
}
