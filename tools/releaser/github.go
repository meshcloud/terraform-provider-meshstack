package main

import (
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	clienthttp "github.com/meshcloud/meshstack-cli/client/http"
)

const commitFields = `... on Commit {
  author { user { login } }
  associatedPullRequests(first: 5) { nodes { number merged body
    closingIssuesReferences(first: 20) { nodes { number
      timelineItems(last: 1, itemTypes: [CLOSED_EVENT]) { nodes { ... on ClosedEvent {
        closer { ... on Commit { oid } }
      } } }
    } }
  } }
}`

type commitNode struct {
	Author struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"author"`
	AssociatedPullRequests struct {
		Nodes []*pullRequest `json:"nodes"`
	} `json:"associatedPullRequests"`
}

type pullRequest struct {
	Number                  int    `json:"number"`
	Merged                  bool   `json:"merged"`
	Body                    string `json:"body"`
	ClosingIssuesReferences struct {
		Nodes []issue `json:"nodes"`
	} `json:"closingIssuesReferences"`
}

type issue struct {
	Number        int `json:"number"`
	TimelineItems struct {
		Nodes []struct {
			Closer struct {
				OID string `json:"oid"`
			} `json:"closer"`
		} `json:"nodes"`
	} `json:"timelineItems"`
}

func (n commitNode) mergedPullRequest() *pullRequest {
	for _, pr := range n.AssociatedPullRequests.Nodes {
		if pr.Merged {
			return pr
		}
	}
	return nil
}

func (i issue) closerCommit() string {
	if events := i.TimelineItems.Nodes; len(events) > 0 {
		return events[0].Closer.OID
	}
	return ""
}

func lookupCommits(shas []string) (map[string]commitNode, error) {
	repository, err := gitHubRepository()
	if err != nil {
		return nil, fmt.Errorf("cannot find the GitHub repository: %w", err)
	}
	token := cmp.Or(os.Getenv("GITHUB_TOKEN"), os.Getenv("GH_TOKEN"))
	if token == "" {
		return nil, errors.New("neither GITHUB_TOKEN nor GH_TOKEN is set, set one, for example with `export GH_TOKEN=$(gh auth token)`")
	}
	owner, name, _ := strings.Cut(repository, "/")
	var query strings.Builder
	fmt.Fprintf(&query, "{ repository(owner: %q, name: %q) {\n", owner, name)
	for _, sha := range shas {
		fmt.Fprintf(&query, "c%s: object(oid: %q) { %s }\n", sha, sha, commitFields)
	}
	query.WriteString("} }")

	type answer struct {
		Data struct {
			Repository map[string]commitNode `json:"repository"`
		} `json:"data"`
		Errors jsontext.Value `json:"errors"`
	}
	got, err := clienthttp.UnmarshalJsonBody[answer](
		clienthttp.NewClient(clienthttp.UserAgent{GitHubRepo: "tools/releaser", Version: "v0"}).
			WithAuthorization(clienthttp.BearerToken(token)).
			DoRequest(context.Background(), clienthttp.MethodPost, &url.URL{Scheme: "https", Host: "api.github.com", Path: "/graphql"},
				clienthttp.WithJsonRequest(map[string]string{"query": query.String()}, ""),
				clienthttp.Retryable(clienthttp.RetryOnTransient),
			))
	if httpErr, ok := errors.AsType[clienthttp.Error](err); ok {
		var body struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(httpErr.ResponseBody, &body)
		return nil, fmt.Errorf("GitHub answered HTTP %d: %s", httpErr.StatusCode, body.Message)
	}
	if err != nil {
		return nil, err
	}
	if got.Data.Repository == nil {
		return nil, fmt.Errorf("GitHub answered without data: %s", got.Errors)
	}
	nodes := make(map[string]commitNode, len(shas))
	for alias, node := range got.Data.Repository {
		nodes[strings.TrimPrefix(alias, "c")] = node
	}
	return nodes, nil
}
