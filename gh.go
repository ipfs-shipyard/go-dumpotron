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
const ghRepo = "protocol/bifrost-infra"

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
	var ghIssue *github.Issue
	//DEBUG
	log.Printf("Searching for issue: %s", name)
	ghIssue, err := searchGHIssue(name)
	if err != nil {
		return ghIssue, fmt.Errorf("failed to fetch Github Issue: %v", err)
	}

	// if ghIssue == nil {
	// // no existing GH issue, create one
		// ghIssue, err := createGHIssue(name)
		// if err != nil {
		// 	return ghIssue, fmt.Errorf("failed to create Github Issue: %v ", err)
		// }
	// }
	return ghIssue, nil
}

func searchGHIssue(name string) (*github.Issue, error){
	//https://github.com/protocol/bifrost-infra/search?q=iSCSI+volume+doesn%27t+unmount&type=Issues
	found, _, err := ghClient.Search.Issues(context.TODO(), "state:open repo:"+ghRepo+" "+name, &github.SearchOptions{
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 1,
		},
	})
	if err != nil {
		return &github.Issue{}, err
	}

	//DEBUG
	//log.Printf("ghClient.Search.Issue(\"%s\") result: %v", name, found)
	if *found.Total == 0 {
		return nil, nil
	}
	return &found.Issues[0], nil
}

func createGHIssue(name string) (*github.Issue, error) {
	return &github.Issue{}, nil
}
