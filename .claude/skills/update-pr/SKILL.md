---
name: update-pr
description: Update GitHub pull request description using REST API to avoid gh pr edit limitations
---

# Update Pull Request Description

Updates a GitHub pull request description using the REST API. This skill works around the `gh pr edit` command limitation caused by Projects (classic) deprecation warnings.

## When to Use This Skill

- When you need to update an existing pull request description
- When `gh pr edit` fails with "Projects (classic) is being deprecated" errors
- When you want to programmatically update PR metadata without using the web UI

## Instructions

Follow these steps to update a PR description:

### Step 1: Parse Input Parameters

Accept the following parameters:
- **PR number** (optional): If not provided, auto-detect from current branch using `gh pr view --json number -q .number`
- **Description content**: Either as direct text or file path

### Step 2: Prepare the Description

If given a file path:
```bash
# Convert markdown file to JSON format
jq -Rs '{body: .}' /path/to/description.md > /tmp/pr_body.json
```

If given direct text:
```bash
# Convert text to JSON format
echo -n "$DESCRIPTION_TEXT" | jq -Rs '{body: .}' > /tmp/pr_body.json
```

### Step 3: Extract Repository Information

Get the repository owner and name from git remote:
```bash
# Get remote URL and extract owner/repo
git remote get-url origin
# Example: git@github.com:owner/repo.git or https://github.com/owner/repo.git
```

Parse the URL to extract:
- Owner name
- Repository name

### Step 4: Update PR Using GitHub API

Use the GitHub REST API to update the pull request:

```bash
gh api \
  --method PATCH \
  -H "Accept: application/vnd.github+json" \
  repos/OWNER/REPO/pulls/PR_NUMBER \
  --input /tmp/pr_body.json
```

### Step 5: Verify the Update

Confirm the update was successful:

```bash
# Fetch and display first 10 lines of updated description
gh pr view PR_NUMBER --json body -q .body | head -10
```

Display confirmation message with:
- PR number
- PR URL: `https://github.com/OWNER/REPO/pull/PR_NUMBER`
- Summary of what was updated

## Examples

### Example 1: Update with file path

```bash
# User provides PR number and file
update-pr 332 /tmp/pr_description.md

# Steps executed:
jq -Rs '{body: .}' /tmp/pr_description.md > /tmp/pr_body.json
gh api --method PATCH repos/koizuka/echonet-list/pulls/332 --input /tmp/pr_body.json
```

### Example 2: Auto-detect PR number

```bash
# User only provides file path, PR number detected from branch
update-pr /tmp/pr_description.md

# Auto-detect PR number:
PR_NUM=$(gh pr view --json number -q .number)
jq -Rs '{body: .}' /tmp/pr_description.md > /tmp/pr_body.json
gh api --method PATCH repos/koizuka/echonet-list/pulls/$PR_NUM --input /tmp/pr_body.json
```

### Example 3: Update with direct text

```bash
# User provides description as text
update-pr 332 "## Summary\n\nThis PR fixes bug #123"

# Convert text to JSON:
echo -n "## Summary\n\nThis PR fixes bug #123" | jq -Rs '{body: .}' > /tmp/pr_body.json
gh api --method PATCH repos/koizuka/echonet-list/pulls/332 --input /tmp/pr_body.json
```

## Error Handling

Handle these common errors:

1. **Invalid PR number**: Verify PR exists before updating
2. **File not found**: Check file path exists before processing
3. **API rate limits**: Report GitHub API rate limit errors clearly
4. **Malformed JSON**: Ensure jq processing succeeds
5. **Permission errors**: User must have write access to repository

## Success Criteria

The skill completes successfully when:
- ✅ PR description is updated on GitHub
- ✅ Verification shows the new description content
- ✅ User receives confirmation with PR URL
- ✅ No errors encountered during API call

## Notes

- This method bypasses the `gh pr edit` Projects (classic) deprecation warning
- The GitHub REST API is more reliable for programmatic updates
- Always verify the update by fetching the PR description after updating
- Temporary files in `/tmp/` are automatically cleaned up by the system
