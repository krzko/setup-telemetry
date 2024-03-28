package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/sethvargo/go-githubactions"
)

const actionName = "set-up-telemetry"

var (
	BUILD_VERSION string
	BUILD_DATE    string
	COMMIT_ID     string
)

func generateTraceID(runID int64, runAttempt int) string {
	return fmt.Sprintf("%d%d", runID, runAttempt)
}

func getGitHubJobName(ctx context.Context, token, owner, repo string, runID, attempt int64) (string, error) {
	splitRepo := strings.Split(repo, "/")
	if len(splitRepo) != 2 {
		return "", fmt.Errorf("GITHUB_REPOSITORY environment variable is malformed: %s", repo)
	}
	owner, repo = splitRepo[0], splitRepo[1]

	client := github.NewClient(nil).WithAuthToken(token)

	opts := &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	runJobs, _, err := client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, opts)
	if err != nil {
		return "", err
	}

	runnerName := os.Getenv("RUNNER_NAME")
	for _, job := range runJobs.Jobs {
		if *job.RunAttempt == attempt && *job.RunnerName == runnerName {
			return *job.Name, nil
		}
	}

	return "", fmt.Errorf("no job found matching the criteria")
}

func main() {
	ctx := context.Background()
	githubactions.Infof("Starting %s version: %s (%s) commit: %s", actionName, BUILD_VERSION, BUILD_DATE, COMMIT_ID)

	if githubactions.GetInput("github-token") == "" {
		githubactions.Fatalf("No GitHub token provided")
	}

	githubToken := githubactions.GetInput("github-token")
	runID, _ := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	runAttempt, _ := strconv.Atoi(os.Getenv("GITHUB_RUN_ATTEMPT"))

	traceID := generateTraceID(runID, runAttempt)
	githubactions.SetOutput("trace-id", traceID)
	githubactions.Infof("Trace ID: %s", traceID)

	jobName, err := getGitHubJobName(ctx, githubToken, os.Getenv("GITHUB_REPOSITORY_OWNER"), os.Getenv("GITHUB_REPOSITORY"), runID, int64(runAttempt))
	if err != nil {
		fmt.Printf("Error getting job name: %v\n", err)
		os.Exit(1)
	}

	githubactions.SetOutput("job-name", jobName)
	githubactions.Infof("Job name: %s", jobName)
}
