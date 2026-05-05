---
name: create-pr
description: Runs Go and Web UI prechecks (fmt/vet/test/build, lint/typecheck/test/build), commits the working changes on a fresh branch with an auto-generated message, pushes, and opens a pull request against main. Use when the user is ready to ship the current uncommitted work as a PR and wants the full pre-flight pipeline run automatically.
allowed-tools: Bash(git add:*), Bash(git status:*), Bash(git diff:*), Bash(git branch:*), Bash(git log:*), Bash(git commit:*), Bash(git checkout:*), Bash(git push:*), Bash(gh pr create:*), Bash(go fmt:*), Bash(go vet:*), Bash(go test:*), Bash(go build:*), Bash(npm run:*), Bash(git rev-parse:*)
---

## Context

- Current git status: !`git status`
- Current git diff (staged and unstaged changes): !`git diff HEAD`
- Current branch: !`git branch --show-current`
- Recent commits: !`git log --oneline -10`
- Web Directory へ移動: !`git rev-parse --show-toplevel` でプロジェクトルートパスを覚えておき、その下の `web` にcdする

## Your Task

以下の作業を自動で実行してください（ユーザーの確認なしで進めてください）:

1. **プリチェック（現在のブランチで実行）**:
   - Go: プロジェクトルートで `go fmt ./...` `go vet ./...` `go test ./...` `go build` を並列実行
   - Web UI: web ディレクトリに移動してから `npm run lint` `npm run typecheck` `npm run test` `npm run build` を並列実行
   - **WebSocket プロトコル変更チェック**:
     - `git diff HEAD` の結果から `protocol/protocol.go` または `server/websocket_server_handlers_*.go` に変更があるか確認
     - 変更がある場合、`docs/websocket_client_protocol.md` も更新されているか確認
     - プロトコルドキュメントが更新されていない場合、ユーザーに警告して確認を求める
   - **ドキュメント更新チェック**:
     - 機能追加・仕様変更があった場合、関連する `docs/*.md`（例: `docs/web_ui_implementation_guide.md`, `docs/console_ui_usage.md`, `docs/internationalization.md` など）が更新されているか確認
     - 更新されていない場合、ユーザーに警告して確認を求める

   ※エラーがあった場合のみ、ユーザーに報告して中断してください。

2. **ブランチ作成とコミット**:
   - 変更内容に基づいて適切なブランチ名を自動生成
   - **ステージング前のファイル確認と選択**:
     1. `git status --porcelain` で対象ファイル一覧を取得
     2. 以下の**機密/不要ブロックリスト**に該当するパスを除外:
        - `.env`, `.env.*`（ただし `.env.example` 等のサンプルは除く）
        - 認証情報・鍵: `*.pem`, `*.key`, `id_rsa*`, `credentials*`, `*.p12`
        - ローカル設定: `.claude/settings.local.json`, `*.local.json`
        - ランタイム/ビルド成果物: `*.log`, `*.pid`, `*.seed`, `coverage/`, `dist/`, `web/bundle/`（ただし意図的な配信物は除く）
        - 巨大バイナリ（数 MB 以上の生成物）
     3. ブロックリスト該当ファイルが含まれていた場合はユーザーに警告して中断（`.gitignore` 漏れの可能性が高い）
     4. 残りのパスを **明示的に列挙して `git add <path1> <path2> ...`** でステージング（`git add .` や `git add -A` は使わない）
   - 変更内容と目的を分析して適切なコミットメッセージを自動生成
   - コミット実行

3. **PR 作成**:
   - ブランチをリモートに push
   - 変更内容を分析して PR 説明を自動生成:
     - 変更の概要（コミット内容から分析）
     - テスト実行結果の確認
   - main ブランチに対する PR 作成
   - PR URL を報告

**重要**: 各ステップでエラーが発生した場合のみユーザーに報告し、成功時は次のステップに自動進行してください。
