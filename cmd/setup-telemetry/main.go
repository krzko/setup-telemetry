package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

	var runJobs *github.Jobs
	var resp *github.Response
	var attempts int = 3 // Number of attempts for retrying API call

	for i := 0; i < attempts; i++ {
		githubactions.Infof("Fetching workflow jobs from GitHub API, attempt %d/%d", i+1, attempts)
		runJobs, resp, err = client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, opts)
		if err == nil {
			runnerName := os.Getenv("RUNNER_NAME")
			found := false
			for _, job := range runJobs.Jobs {
				if job.RunnerName != nil && job.Name != nil && job.RunAttempt != nil {
					if *job.RunAttempt == attempt && *job.RunnerName == runnerName {
						createdAt = job.CreatedAt.Format(time.RFC3339)
						startedAt = job.StartedAt.Format(time.RFC3339)
						githubactions.Infof("Match found, job name: %s", *job.Name)
						return strconv.FormatInt(*job.ID, 10), *job.Name, createdAt, startedAt, nil
					}
				}
			}
			if !found {
				githubactions.Infof("No matching job found on attempt %d", i+1)
				if i < attempts-1 { // Retry if the maximum number of attempts is not reached
					time.Sleep(3 * time.Second) // Wait
					continue
				}
			}
		} else {
			githubactions.Errorf("Failed to fetch workflow jobs: %v", err)
			if resp != nil {
				githubactions.Infof("GitHub API response status: %s", resp.Status)
			}
			if i < attempts-1 { // Retry if the maximum number of attempts is not reached
				time.Sleep(3 * time.Second) // Wait
				continue
			}
		}
		break
	}

	// Generate a fallback job span ID if no job is found after all attempts
	spanID, genErr := generateJobSpanID(runID, int(attempt), "fallback-job")
	if genErr != nil {
		githubactions.Errorf("Error generating fallback job span ID: %v", genErr)
		return "", "", "", "", fmt.Errorf("failed to retrieve job and generate fallback span ID: %v", genErr)
	}
	return "", "fallback-job", "", "", fmt.Errorf("no job found matching the criteria after %d attempts, fallback span ID: %s", attempts, spanID)
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
