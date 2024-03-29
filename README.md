# Set Up Telemetry Action

This action exports trace and job IDs, and sets up a traceparent for use in GitHub Actions workflows to enable telemetry and tracing.

It is important to note that `job-id` and `job-name` are not directly accessible through the [GitHub environment variables](https://docs.github.com/en/actions/learn-github-actions/variables).

The `job-name` output uses the default `name` or the name provided by `run-name`, rather than the rendered name that might be applied by a composite workflow or matrix strategy.

This action provides a consistent and accessible method to retrieve these values for further use in your workflow, which is especially useful in complex workflows where these values need to be explicitly managed or passed between jobs.

Trace IDs and job span IDs are generated in a deterministic fashion from the associated run ID and run attempt, with job span IDs also incorporating the job name. This deterministic generation ensures consistent and predictable identifiers for tracing and telemetry across workflow executions.

## Usage

### Pre-requisites

Create a workflow `.yml` file in your repository's `.github/workflows` directory. For more information, see the GitHub Help Documentation for [Creating a workflow file](https://help.github.com/en/articles/configuring-a-workflow#creating-a-workflow-file).

### Inputs

- `github-token`: A token that can be used with the GitHub API. Default is `${{ github.token }}`.

### Outputs

- `trace-id`: The generated trace ID for the workflow run.
- `job-id`: The ID of the GitHub Actions job.
- `job-name`: The name of the GitHub Actions job.
- `job-span-id`: The generated span ID for the job.
- `traceparent`: The W3C Trace Context traceparent value for the workflow run.

### Example Usage

```yaml
name: Example Telemetry Workflow

on: [push]

jobs:
  telemetry:
    runs-on: ubuntu-latest

    steps:
    - name: Checkout
      uses: actions/checkout@v4

    - name: Set up telemetry
      id: set-up-telemetry
      uses: krzko/set-up-telemetry@v0.1.0

    - name: Use Trace ID
      run: echo "Trace ID: ${{ steps.set-up-telemetry.outputs.trace-id }}"

    - name: Use Job Span ID
      run: echo "Job Span ID: ${{ steps.set-up-telemetry.outputs.job-span-id }}"

    - name: Use Traceparent
      run: echo "Traceparent: ${{ steps.set-up-telemetry.outputs.traceparent }}"
```

In this workflow, the `Set up telemetry` action is used to generate and output the trace ID, job ID, job name, job span ID, and traceparent, which can then be used in subsequent steps of the workflow.

### Contributing

Contributions to `krzko/set-up-telemetry` are welcome! Please refer to the repository's CONTRIBUTING.md for guidelines on how to submit contributions.

## License

The scripts and documentation in this project are released under the Apache License.
