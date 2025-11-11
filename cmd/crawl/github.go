package main

import (
	"context"
	"time"

	"github.com/anthrotech-dev/activity"
	"github.com/google/go-github/v77/github"
)

func collectGitHub(ctx context.Context) ([]activity.Activity, error) {
	var activities []activity.Activity

	client := github.NewClient(nil)

	owner := "anthrotech-dev"
	repo := "anthrotech-dev"

	jst, _ := time.LoadLocation("Asia/Tokyo")
	nowJST := time.Now().In(jst)
	since := time.Date(nowJST.Year(), nowJST.Month(), nowJST.Day()-2, 0, 0, 0, 0, jst)
	until := time.Date(nowJST.Year(), nowJST.Month(), nowJST.Day()-1, 23, 59, 59, 999000000, jst)

	commits, _, err := client.Repositories.ListCommits(
		ctx,
		owner,
		repo,
		&github.CommitsListOptions{
			Since: since,
			Until: until,
		},
	)
	if err != nil {
		return nil, err
	}

	for _, commit := range commits {
		act := activity.Activity{
			ID:        "anthrotech-dev-" + commit.GetSHA(),
			UserID:    *commit.Author.Login,
			Timestamp: commit.Commit.Committer.Date.Time,
			Type:      "GitHub",
			Body:      commit.GetCommit().GetMessage(),
		}
		activities = append(activities, act)
	}

	return activities, nil
}
