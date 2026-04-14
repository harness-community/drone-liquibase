## JIRA Ticket

**Ticket:** [DBOPS-XXXXX](https://harness.atlassian.net/browse/DBOPS-XXXXX)

## Describe your changes

Description...

## Code Author Checklist

[Code Author Guidelines](https://harness.atlassian.net/wiki/spaces/ENGOPS/pages/21149417486/Code+Review+Guidelines#Code-Author-Guidelines)

- Tested the changes in the Local/Devspace environment.
- Ran `go build && go test ./...` before committing.
- Tested Docker image builds for affected variants (standard, mongo, spanner, snowflake).


## Code Reviewer Checklist

[Code Reviewer Guidelines](https://harness.atlassian.net/wiki/spaces/ENGOPS/pages/21149417486/Code+Review+Guidelines#Code-Reviewer-Guidelines)
- Title & description clearly summarize changes and context. [Link](https://harness.atlassian.net/wiki/spaces/ENGOPS/pages/21149417486/Code+Review+Guidelines#Write-Good-PR-Descriptions)
- Reviewed for correctness, missing changes to related features and security issues. [Link](https://harness.atlassian.net/wiki/spaces/ENGOPS/pages/21149417486/Code+Review+Guidelines#What-to-Look-For%3F)
- Reviewed for performance impact on production systems.
- Reviewed for deployment safety (forward/backward compatibility, rollback safety, no need for multiple services to be deployed together).
- Code is readable, maintainable with adequate documentation (comments, README, etc.). [Link](https://harness.atlassian.net/wiki/spaces/ENGOPS/pages/21149417486/Code+Review+Guidelines#Write-for-the-Future-%3A-Code-Maintainability)
-  Automated tests (unit and/or integration) cover all changes.
- Logs added with appropriate log levels for debugging and troubleshooting.


Communication & Documentation

-  (If applicable - Behavior Change): Product Manager and Customer Success Team are informed at least 15 days prior for customer notification.
-  (If applicable - Release Notes Required): JIRA ticket is marked as "Release Notes Candidate" and Release Notes are added in the JIRA.
-  (If applicable - Documentation Update Required): DOC Update JIRA ticket is linked in the Pull Request description.

## [Latest PR Check Triggers]

<details>
  <summary>PR Check triggers</summary>

- Unit Tests: `trigger tests`
- MessageMetadata: `trigger messagecheck`
- Lint Check: `trigger lint`
- Build Check: `trigger build`
</details>
