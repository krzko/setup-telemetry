# Setup Telemetry Action

This action exports trace and job IDs, and sets up a traceparent for use in GitHub Actions workflows to enable telemetry and tracing. It is intended to be used in conjunction with the [OpenTelemetry Collector GitHub Actions Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/27460). This receiver processes GitHub Actions webhook events to observe workflows and jobs, converting them into trace telemetry for detailed observability.

It is important to note that `job-id` and `job-name` are not directly accessible through the [GitHub environment variables](https://docs.github.com/en/actions/learn-github-actions/variables).

The `job-name` output uses the default `name` or the name provided by `run-name`, rather than the rendered name that might be applied by a composite workflow or matrix strategy.

This action provides a consistent and accessible method to retrieve these values for further use in your workflow, which is especially useful in complex workflows where these values need to be explicitly managed or passed between jobs.

Trace IDs and job span IDs are generated in a deterministic fashion from the associated run ID and run attempt, with job span IDs also incorporating the job name. This deterministic generation ensures consistent and predictable identifiers for tracing and telemetry across workflow executions.

## GitHub Actions Receiver

The GitHub Actions Receiver processes GitHub Actions webhook events to observe workflows and jobs. It handles `workflow_job` and `workflow_run` event payloads, transforming them into trace telemetry. This allows the observation of workflow execution times, success, and failure rates. If a secret is configured (recommended), it validates the payload ensuring data integrity before processing.

## Usage

### Pre-requisites

Create a workflow `.yml` file in your repository's `.github/workflows` directory. For more information, see the GitHub Help Documentation for [Creating a workflow file](https://help.github.com/en/articles/configuring-a-workflow#creating-a-workflow-file).

### Inputs

- `github-token`: A token that can be used with the GitHub API. Default is `${{ github.token }}`.

### Outputs

- `created-ad`" The timestamp when the workflow run was created.
- `job-id`: The ID of the GitHub Actions job.
- `job-name`: The name of the GitHub Actions job.
- `job-span-id`: The generated span ID for the job.
- `started-at`: The timestamp when the workflow run started.
- `trace-id`: The generated trace ID for the workflow run.
- `traceparent`: The W3C Trace Context traceparent value for the workflow run.

### Example Usage

```yaml
name: Test and Build

on:
  push:

env:
  otel-exporter-otlp-endpoint: otelcol.foo.corp:443
  otel-service-name: o11y.workflows
  otel-resource-attributes: deployment.environent=dev,service.version=0.1.0

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Setup telemetry
        id: setup-telemetry
        uses: krzko/setup-telemetry@v0.4.1

      - name: Checkout
        uses: actions/checkout@v4

      - run: # do_some_work

      - name: Export job telemetry
        if: always()
        uses: krzko/export-job-telemetry@v0.4.1
        with:
          created-at: ${{ steps.setup-telemetry.outputs.created-at }}
          job-status: ${{ job.status }}
          job-name: ${{ steps.setup-telemetry.outputs.job-name }}
          otel-exporter-otlp-endpoint: ${{ env.otel-exporter-otlp-endpoint }}
          otel-resource-attributes: "foo.new_attribute=123,${{ env.otel-resource-attributes }}"
          otel-service-name: ${{ env.otel-service-name }}
          started-at: ${{ steps.setup-telemetry.outputs.started-at }}
          traceparent: ${{ steps.setup-telemetry.outputs.traceparent }}

  build:
    runs-on: ubuntu-latest
    steps:
      - name: Setup telemetry
        id: setup-telemetry
        uses: krzko/setup-telemetry@v0.4.1

      - name: Checkout
        uses: actions/checkout@v4

      - run: # do_some_work

      - name: Export job telemetry
        if: always()
        uses: krzko/export-job-telemetry@v0.4.1
        with:
          created-at: ${{ steps.setup-telemetry.outputs.created-at }}
          job-status: ${{ job.status }}
          job-name: ${{ steps.setup-telemetry.outputs.job-name }}
          otel-exporter-otlp-endpoint: ${{ env.otel-exporter-otlp-endpoint }}
          otel-resource-attributes: "foo.new_attribute=123,${{ env.otel-resource-attributes }}"
          otel-service-name: ${{ env.otel-service-name }}
          started-at: ${{ steps.setup-telemetry.outputs.started-at }}
          traceparent: ${{ steps.setup-telemetry.outputs.traceparent }}
```

In this workflow, the `Setup telemetry` action is used to generate and output the trace ID, job ID, job name, job span ID, and traceparent, which can then be used in subsequent steps of the workflow, such as using the [krzko/export-job-telemetry](https://github.com/krzko/export-job-telemetry) action.

### Contributing

Contributions to `krzko/setup-telemetry` are welcome! Please refer to the repository's CONTRIBUTING.md for guidelines on how to submit contributions.

## License

The scripts and documentation in this project are released under the Apache License.
