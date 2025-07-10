---
description: "Clean up after PR merge - switch to main branch and update"
allowed-tools: ["Bash"]
---

# PR Merge Cleanup

Execute the following commands to clean up after a PR has been merged.

If no branch name is provided as an argument, I will use the current branch name and ask for confirmation before proceeding.

Commands to execute:
1. Switch to main branch
2. Pull latest changes from origin
3. Delete the merged branch (locally)
4. Prune remote tracking branches

!git status
!echo "Current branch will be deleted after switching to main. Proceed? (y/n)"
!read -p "Continue? " -n 1 -r
!echo
!if [[ $REPLY =~ ^[Yy]$ ]]; then
!  BRANCH_TO_DELETE=${ARGUMENTS:-$(git branch --show-current)}
!  echo "Switching to main branch..."
!  git checkout main
!  echo "Pulling latest changes..."
!  git pull origin main
!  echo "Deleting branch: $BRANCH_TO_DELETE"
!  git branch -d $BRANCH_TO_DELETE
!  echo "Pruning remote tracking branches..."
!  git remote prune origin
!  echo "Cleanup complete!"
!else
!  echo "Operation cancelled."
!fi