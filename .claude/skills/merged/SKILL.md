---
name: merged
description: Cleans up the local repository after a pull request has been merged on GitHub by switching to main, pulling the latest changes, deleting the merged branch locally, and pruning stale remote tracking refs. Use after a PR has been merged on GitHub and the working branch should be reclaimed.
allowed-tools: Bash(git checkout:*), Bash(git pull:*), Bash(git remote prune:*), Bash(git branch:*), Bash(git status:*), Bash(git log:*)
---

# PR Merge Cleanup

Clean up after a PR has been merged. This skill will:

1. Switch to main branch
2. Pull latest changes from origin
3. Delete the merged branch locally
4. Prune remote tracking branches

## Context

- Current branch: !`git branch --show-current`

## Your task

Perform post-merge cleanup for the branch: ${ARGUMENTS:-$(git branch --show-current)}

Please execute the following steps immediately without confirmation:
1. Save the current branch name
2. Switch to main branch
3. Pull latest changes from origin/main
4. Delete the merged branch locally (use -D since GitHub auto-deletes merged branches)
5. Prune remote tracking branches
6. Show a summary of what was cleaned up
