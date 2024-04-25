package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/sethvargo/go-githubactions"
)

const actionName = "setup-telemetry"

var (
	BUILD_VERSION string
	BUILD_DATE    string
	COMMIT_ID     string
)

func generateTraceID(runID int64, runAttempt int) string {
	input := fmt.Sprintf("%d%dt", runID, runAttempt)
	hash := sha256.Sum256([]byte(input))
	traceIDHex := hex.EncodeToString(hash[:])
	traceID := traceIDHex[:32]
	return traceID
}

func generateJobSpanID(runID int64, runAttempt int, job string) (string, error) {
	input := fmt.Sprintf("%d%d%s", runID, runAttempt, job)
	hash := sha256.Sum256([]byte(input))
	spanIDHex := hex.EncodeToString(hash[:])
	spanID := spanIDHex[16:32]
	return spanID, nil
}

func getGitHubJobInfo(ctx context.Context, token, owner, repo string, runID, attempt int64) (jobID, jobName, createdAt, startedAt string, err error) {
	splitRepo := strings.Split(repo, "/")
	if len(splitRepo) != 2 {
		return "", "", "", "", fmt.Errorf("GITHUB_REPOSITORY environment variable is malformed: %s", repo)
	}
	owner, repo = splitRepo[0], splitRepo[1]

	client := github.NewClient(nil).WithAuthToken(token)
	opts := &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	githubactions.Infof("Fetching workflow jobs from GitHub API")
	runJobs, resp, err := client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, opts)
	if err != nil {
		githubactions.Errorf("Failed to fetch workflow jobs: %v", err)
		if resp != nil {
			githubactions.Infof("GitHub API response status: %s", resp.Status)
			if resp.Body != nil {
				bodyBytes, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr != nil {
					githubactions.Errorf("Failed to read response body: %v", readErr)
				} else {
					githubactions.Debugf("GitHub API response body: %s", string(bodyBytes))
				}
			}
		}
		return "", "", "", "", err
	}

	runnerName := os.Getenv("RUNNER_NAME")
	githubactions.Infof("Looking for job with runner name: %s", runnerName)
	for _, job := range runJobs.Jobs {
		if job.RunnerName != nil && job.Name != nil && job.RunAttempt != nil {
			githubactions.Infof("Inspecting job: %s, runner: %s, attempt: %d", *job.Name, *job.RunnerName, *job.RunAttempt)
			if *job.RunAttempt == attempt && *job.RunnerName == runnerName {
				githubactions.Infof("Match found, job name: %s", *job.Name)
				createdAt = job.CreatedAt.Format(time.RFC3339)
				startedAt = job.StartedAt.Format(time.RFC3339)
				return strconv.FormatInt(*job.ID, 10), *job.Name, createdAt, startedAt, nil
			}
		}
	}

	err = fmt.Errorf("no job found matching the criteria")
	githubactions.Errorf("Error: %v", err)
	return "", "", "", "", fmt.Errorf("no job found matching the criteria")
}

func main() {
	ctx := context.Background()
	githubactions.Infof("Starting %s version: %s (%s) commit: %s", actionName, BUILD_VERSION, BUILD_DATE, COMMIT_ID)

	githubToken := githubactions.GetInput("github-token")
	if githubToken == "" {
		githubactions.Fatalf("No GitHub token provided")
	}

	runID, _ := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	runAttempt, _ := strconv.Atoi(os.Getenv("GITHUB_RUN_ATTEMPT"))

	traceID := generateTraceID(runID, runAttempt)
	githubactions.SetOutput("trace-id", traceID)
	githubactions.Infof("trace-id: %s", traceID)

	jobID, jobName, createdAt, startedAt, err := getGitHubJobInfo(ctx, githubToken, os.Getenv("GITHUB_REPOSITORY_OWNER"), os.Getenv("GITHUB_REPOSITORY"), runID, int64(runAttempt))
	if err != nil {
		githubactions.Errorf("Error getting job info: %v", err)
		os.Exit(1)
	}

	githubactions.SetOutput("job-id", jobID)
	githubactions.Infof("job-id: %s", jobID)
	githubactions.SetOutput("job-name", jobName)
	githubactions.Infof("job-name: %s", jobName)
	githubactions.SetOutput("created-at", createdAt)
	githubactions.Infof("created-at: %s", createdAt)
	githubactions.SetOutput("started-at", startedAt)
	githubactions.Infof("started-at: %s", startedAt)

	jobSpanID, err := generateJobSpanID(runID, int(runAttempt), jobName)
	if err != nil {
		githubactions.Errorf("Error generating job span ID: %v", err)
		os.Exit(1)
	}

	githubactions.SetOutput("job-span-id", jobSpanID)
	githubactions.Infof("job-span-id: %s", jobSpanID)

	traceparent := fmt.Sprintf("00-%s-%s-01", traceID, jobSpanID)
	githubactions.SetOutput("traceparent", traceparent)
	githubactions.Infof("traceparent: %s", traceparent)

	markdownSummary := fmt.Sprintf("### 🚦 %s\n", actionName)
	markdownSummary += fmt.Sprintf("trace-id: `%s`\n", traceID)
	markdownSummary += fmt.Sprintf("traceparent: `%s`\n", traceparent)

	observabilityBackendURL := githubactions.GetInput("observability-backend-url")
	if observabilityBackendURL != "" {
		traceLink := fmt.Sprintf("%s%s", observabilityBackendURL, traceID)
		markdownSummary += fmt.Sprintf("\n🔗 [View trace](%s)\n", traceLink)
		githubactions.Infof("trace-link: %s", traceLink)
		githubactions.SetOutput("trace-link", traceLink)
	}

	githubactions.AddStepSummary(markdownSummary)
}
