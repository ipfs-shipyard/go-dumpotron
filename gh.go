package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
	log "github.com/sirupsen/logrus"
)

var ghClient *github.Client
const ghRepo = "bifrost-infra"
const ghOwner = "protocol"
const ghOwnerRepo =  ghOwner + "/" + ghRepo

func setupGHClient(authToken string){
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: authToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	ghClient = github.NewClient(tc)
	log.Debugf("Created new go-github Client: %v", ghClient)

}

func getGHIssue(name string) (*github.Issue, error) {
	log.Debugf("Searching for issue: %s", name)
	ghIssue, err := searchGHIssue(name)
	if err != nil {
		return ghIssue, fmt.Errorf("failed to fetch Github Issue: %v", err)
	}

	if ghIssue != nil {
		return ghIssue, nil
	}

	// no existing GH issue, create one
	log.Infof("Issue not found: %s", name)
	ghIssue, err = createGHIssue(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create Github Issue: %v ", err)
	}
	return ghIssue, nil
}

func searchGHIssue(name string) (*github.Issue, error){
	//https://github.com/protocol/bifrost-infra/search?q=iSCSI+volume+doesn%27t+unmount&type=Issues
	found, _, err := ghClient.Search.Issues(context.TODO(), "state:open repo:"+ghOwnerRepo+" "+name, &github.SearchOptions{
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Search.Issues returned error: %v", err)
	}

	log.Debugf("ghClient.Search.Issue(\"%s\") result: %v", name, found)
	if *found.Total == 0 {
		return nil, nil
	}
	return &found.Issues[0], nil
}

func createGHIssue(name string) (*github.Issue, error) {
	log.Debugf("Creating issue: %s", name)
	body := fmt.Sprintf("Tracking pprof dumps for go-ipfs version %s", name)
	issueRequest := &github.IssueRequest{
			Title:    &name,
			Body:     &body,
		}

	issue, _, err := ghClient.Issues.Create(context.Background(), ghOwner, ghRepo, issueRequest)
	if err != nil {
		return nil, fmt.Errorf("Issues.Create returned error: %v", err)
	}

	return issue, err
}

func postArchiveCIDtoGH(cidURL string, issue *github.Issue) (string, error) {
	log.Debugf("Adding comment to issue #%d: %s", *issue.Number, cidURL)
	input := &github.IssueComment{Body: &cidURL}
	comment, _, err := ghClient.Issues.CreateComment(context.Background(), ghOwner, ghRepo, *issue.Number, input)
	if err != nil {
		return "", fmt.Errorf("Issues.CreateComment returned error: %v", err)
	}
	return *comment.HTMLURL, nil
}
