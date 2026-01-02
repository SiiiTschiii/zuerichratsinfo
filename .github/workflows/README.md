# GitHub Actions Workflows

## Run a Workflow Locally

To test the validate-contacts workflow locally (without committing):

```bash
act -W .github/workflows/validate-contacts.yml --env GITHUB_EVENT_NAME=pull_request
```
