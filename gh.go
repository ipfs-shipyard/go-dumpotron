package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v29/github"
	// "net/http"
	"golang.org/x/oauth2"
	"log"
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
	//DEBUG
	//log.Printf("Created new go-github Client: %v", ghClient)

}

func getGHIssue(name string) (*github.Issue, error) {
	//DEBUG
	log.Printf("Searching for issue: %s", name)
	ghIssue, err := searchGHIssue(name)
	if err != nil {
		return ghIssue, fmt.Errorf("failed to fetch Github Issue: %v", err)
	}

	if ghIssue != nil {
		return ghIssue, nil
	}

	// no existing GH issue, create one
	//DEBUG
	log.Printf("Issue not found: %s", name)
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

	//DEBUG
	//log.Printf("ghClient.Search.Issue(\"%s\") result: %v", name, found)
	if *found.Total == 0 {
		return nil, nil
	}
	return &found.Issues[0], nil
}

func createGHIssue(name string) (*github.Issue, error) {
	//DEBUG
	log.Printf("Creating issue: %s", name)
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
