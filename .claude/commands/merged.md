---
description: "Clean up after PR merge - switch to default branch and update"
allowed-tools: ["Bash"]
---

# PR Merge Cleanup

Clean up after a PR has been merged. This command will:

1. Detect the default branch dynamically (main/master)
2. Switch to the default branch
3. Pull latest changes from origin
4. Delete the merged branch locally (with confirmation)
5. Prune remote tracking branches

## Context

- Current branch: !`git branch --show-current`
- Repository status: !`git status --porcelain`
- Available branches: !`git branch`

## Your task

Perform post-merge cleanup for the branch: ${ARGUMENTS:-$(git branch --show-current)}

Please:
1. Validate the branch name for security (only alphanumeric, hyphens, underscores, slashes)
2. Dynamically detect the default branch (main/master)
3. Ensure we're not trying to delete the default branch
4. Confirm the branch exists locally
5. Ask for user confirmation before proceeding
6. Execute the cleanup steps with proper error handling
7. Handle the case where branch deletion fails (offer force delete option)