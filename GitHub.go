package main

import (
	"context"
	"os"
	"strings"
	"net/url"
	"encoding/base64"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func UpdateFileAndCommit(content string) error {
	apiToken := os.Getenv("API_TOKEN")
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	opmlUrl := os.Getenv("FEEDS_URL")
	parsedUrl, err := url.Parse(opmlUrl)
	if err != nil {
		return err
	}
	segments := strings.Split(parsedUrl.Path, "/")
	username := segments[1]
	repoName := segments[2]
	branchName := segments[3]
	filePath := strings.Join(segments[4:], "/")

	var query struct {
    Repository struct {
			Ref struct {
				Target struct {
					Commit struct {
						Oid string
					} `graphql:"... on Commit"`
				}
			} `graphql:"ref(qualifiedName: $qualifiedName)"`
    } `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":           githubv4.String(username),
		"name":            githubv4.String(repoName),
		"qualifiedName":   githubv4.String(branchName),
	}
	err = client.Query(context.Background(), &query, variables)
	if err != nil {
		return err
	}
	commitOid := query.Repository.Ref.Target.Commit.Oid

	var mutation struct {
		CreateCommitOnBranch struct {
			ClientMutationId string
		} `graphql:"createCommitOnBranch(input: $input)"`
	}
	repo := githubv4.String(username + "/" + repoName)
	branch := githubv4.String(branchName)
	input := githubv4.CreateCommitOnBranchInput{
		Branch: githubv4.CommittableBranch{
			RepositoryNameWithOwner: &repo,
			BranchName:              &branch,
		},
		Message: githubv4.CommitMessage{
			Headline: githubv4.String("Update " + filePath),
		},
		FileChanges: &githubv4.FileChanges{
			Additions: &[]githubv4.FileAddition{
				{
					Path:     githubv4.String(filePath),
					Contents: githubv4.Base64String(base64.StdEncoding.EncodeToString([]byte(content))),
				},
			},
		},
		ExpectedHeadOid: githubv4.GitObjectID(commitOid),
	}
	err = client.Mutate(context.Background(), &mutation, input, nil)
	return err
}
