---
allowed-tools: Bash(git add:*), Bash(git status:*), Bash(git commit:*), Bash(git checkout:*), Bash(git push:*), Bash(gh pr create:*), Bash(go fmt:*), Bash(go vet:*), Bash(go test:*), Bash(go build:*), Bash(npm run:*)
Description: create a pull request
---

## Context

- Current git status: !`git status`
- Current git diff (staged and unstaged changes): !`git diff HEAD`
- Current branch: !`git branch --show-current`
- Recent commits: !`git log --oneline -10`

## Your Task

1. コミット前のチェックを実行してください：
   - Go (プロジェクトルートで実行): `go fmt ./...`, `go vet ./...`, `go test ./...`, `go build`
   - Web UI (webディレクトリで実行): `cd $(git rev-parse --show-toplevel)/web && npm run lint && npm run typecheck && npm run test && npm run build`

2. 新しいブランチを作成し、現在の変更をコミットします。コミットメッセージは変更内容と目的を簡潔に説明してください。

3. ブランチをリモートにpushし、mainブランチに対するpull requestを作成してください。PR説明には：
   - 変更の概要
   - テスト方法
   - 関連するissue番号（もしあれば）

