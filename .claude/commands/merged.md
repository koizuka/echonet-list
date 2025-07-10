---
description: "Clean up after PR merge - switch to default branch and update"
allowed-tools: ["Bash"]
---

# PR Merge Cleanup

Execute the following commands to clean up after a PR has been merged.

If no branch name is provided as an argument, I will use the current branch name and ask for confirmation before proceeding.

Commands to execute:
1. Detect default branch (main/master)
2. Switch to default branch
3. Pull latest changes from origin
4. Delete the merged branch (locally)
5. Prune remote tracking branches

!set -euo pipefail
!
!# Function to validate branch name
!validate_branch_name() {
!  local branch="$1"
!  if [[ ! "$branch" =~ ^[a-zA-Z0-9_/-]+$ ]]; then
!    echo "Error: Invalid branch name '$branch'. Only alphanumeric characters, hyphens, underscores, and slashes are allowed."
!    exit 1
!  fi
!  if [[ "$branch" == "." || "$branch" == ".." ]]; then
!    echo "Error: Invalid branch name '$branch'."
!    exit 1
!  fi
!}
!
!# Function to get default branch
!get_default_branch() {
!  local default_branch
!  default_branch=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || echo "")
!  if [[ -z "$default_branch" ]]; then
!    # Try to determine from common branch names
!    if git show-ref --verify --quiet refs/remotes/origin/main; then
!      default_branch="main"
!    elif git show-ref --verify --quiet refs/remotes/origin/master; then
!      default_branch="master"
!    else
!      echo "Error: Cannot determine default branch. Please run 'git remote set-head origin -a' first."
!      exit 1
!    fi
!  fi
!  echo "$default_branch"
!}
!
!# Get current branch if no argument provided
!CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "")
!if [[ -z "$CURRENT_BRANCH" ]]; then
!  echo "Error: Not on a branch (detached HEAD state)."
!  exit 1
!fi
!
!BRANCH_TO_DELETE="${ARGUMENTS:-$CURRENT_BRANCH}"
!
!# Validate branch name
!validate_branch_name "$BRANCH_TO_DELETE"
!
!# Get default branch
!DEFAULT_BRANCH=$(get_default_branch)
!
!# Check if trying to delete the default branch
!if [[ "$BRANCH_TO_DELETE" == "$DEFAULT_BRANCH" ]]; then
!  echo "Error: Cannot delete the default branch '$DEFAULT_BRANCH'."
!  exit 1
!fi
!
!# Check if branch exists
!if ! git show-ref --verify --quiet "refs/heads/$BRANCH_TO_DELETE"; then
!  echo "Error: Branch '$BRANCH_TO_DELETE' does not exist."
!  exit 1
!fi
!
!# Show status and confirm
!echo "Current repository status:"
!git status --porcelain
!echo
!echo "Default branch: $DEFAULT_BRANCH"
!echo "Branch to delete: $BRANCH_TO_DELETE"
!echo "Current branch: $CURRENT_BRANCH"
!echo
!echo "This will:"
!echo "1. Switch to '$DEFAULT_BRANCH' branch"
!echo "2. Pull latest changes from origin"
!echo "3. Delete branch '$BRANCH_TO_DELETE' locally"
!echo "4. Prune remote tracking branches"
!echo
!
!# Use printf instead of read for better compatibility
!printf "Proceed with cleanup? (y/N): "
!read -r REPLY
!echo
!
!if [[ "$REPLY" =~ ^[Yy]$ ]]; then
!  echo "Starting cleanup..."
!  
!  # Switch to default branch
!  echo "Switching to '$DEFAULT_BRANCH' branch..."
!  if ! git checkout "$DEFAULT_BRANCH"; then
!    echo "Error: Failed to switch to '$DEFAULT_BRANCH' branch."
!    exit 1
!  fi
!  
!  # Pull latest changes
!  echo "Pulling latest changes..."
!  if ! git pull origin "$DEFAULT_BRANCH"; then
!    echo "Error: Failed to pull latest changes."
!    exit 1
!  fi
!  
!  # Delete the branch
!  echo "Deleting branch '$BRANCH_TO_DELETE'..."
!  if ! git branch -d "$BRANCH_TO_DELETE"; then
!    echo "Warning: Failed to delete branch '$BRANCH_TO_DELETE' with -d flag."
!    printf "Force delete with -D? (y/N): "
!    read -r FORCE_REPLY
!    if [[ "$FORCE_REPLY" =~ ^[Yy]$ ]]; then
!      if ! git branch -D "$BRANCH_TO_DELETE"; then
!        echo "Error: Failed to force delete branch '$BRANCH_TO_DELETE'."
!        exit 1
!      fi
!      echo "Branch '$BRANCH_TO_DELETE' force deleted."
!    else
!      echo "Branch deletion cancelled."
!      exit 1
!    fi
!  else
!    echo "Branch '$BRANCH_TO_DELETE' deleted successfully."
!  fi
!  
!  # Prune remote tracking branches
!  echo "Pruning remote tracking branches..."
!  if ! git remote prune origin; then
!    echo "Warning: Failed to prune remote tracking branches."
!  fi
!  
!  echo "Cleanup complete!"
!else
!  echo "Operation cancelled."
!fi