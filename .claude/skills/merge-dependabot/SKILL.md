---
name: merge-dependabot
description: Categorizes open dependabot PRs into safe (patch/minor) versus needs-review (major bumps, Go deps), enables auto-merge on the safe ones, and shepherds the merges by handling conflicts and CI failures until none remain. Use when there is a backlog of dependabot PRs to clear and the user wants the safe ones merged automatically.
allowed-tools: Bash(gh pr list:*), Bash(gh pr view:*), Bash(gh pr merge:*), Bash(gh pr comment:*), Bash(gh pr checks:*), Bash(git pull)
---

Merge all safe dependabot PRs efficiently and automatically:

## Process

1. **List PRs**:

   ```bash
   gh pr list --author app/dependabot --json number,title,autoMergeRequest,mergeStateStatus
   ```

2. **Categorize PRs**:
   - **Already auto-merge**: PRs with `autoMergeRequest` set were enabled by the `dependabot-auto-merge.yml` workflow (build tooling, a few low-risk libraries, official CI actions; patch/minor only). Don't re-categorize them; just monitor them in step 6.
   - **Safe**: Patch/minor updates of known safe packages (see Safety criteria below)
   - **Needs review**: Major version updates, React updates, unknown packages

3. **Ask user about non-safe PRs**: If there are major version updates or other risky PRs, ask user whether to include them

4. **Create todo list**: Create a todo list with approved PRs to track progress

5. **Set auto-merge on all approved PRs at once**:

   ```bash
   echo "<approved PR numbers>" | xargs -n1 gh pr merge --squash --auto
   ```

   No kick-off is needed: PRs merge in parallel as their CI passes (see Notes).

6. **Monitor until all PRs are processed**:
   - Check every 30-60 seconds: `gh pr list --author app/dependabot --json number,state,mergeable,mergeStateStatus,statusCheckRollup`
   - Handle issues as they arise:
     - **Conflicts** (`DIRTY`, usually `web/package-lock.json` after another PR merged): Dependabot normally rebases on its own within a few minutes. If not, `gh pr comment <PR#> --body "@dependabot rebase"`; use `@dependabot recreate` only if rebase fails.
     - **CI failures**: Disable auto-merge on that PR (`gh pr merge --disable-auto <PR#>`), report to user, continue with remaining PRs
     - **Closed by dependabot**: Normal when the dependency was superseded or removed; drop it from the todo list
   - If a PR is stuck for >10 minutes, investigate manually
   - Continue until no approved PRs remain open

7. **Verify completion**: `gh pr list --author app/dependabot` should show no results (or only skipped PRs). Check that the CI run on `main` after the last merge passed, since PRs were not tested on top of each other.

8. **Sync local repository**: If on main branch, run `git pull` to update with merged changes

## Safety criteria

Focus on dependency updates with patch/minor version bumps. Use your knowledge of common packages to determine safety.

Skip or ask user about:

- Major version updates (e.g., 1.x → 2.x)
- Go dependency updates
- Updates with failing CI

## Notes

- The `main` ruleset requires the `CI Summary` check but does **not** require branches to be up to date with `main` (`strict_required_status_checks_policy: false`). So PRs don't need to be merged one by one or have their branch updated before merging.
- The trade-off: two PRs can each pass CI but break `main` in combination. That's why step 7 checks the `main` CI run.
