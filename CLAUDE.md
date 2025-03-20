# Claude Instructions

## Project Context
- このプロジェクトはECHONET Liteプロトコルを扱うGoプログラムです
- .clinerules/ ディレクトリに開発指示が含まれています
- Claude使用時は、まず以下のファイルを確認してください:
  - `.clinerules/00index.md` - ファイルの役割と重複回避の説明
  - `.clinerules/01Overview.md` - プロジェクト全体の技術的な概要
  - `.clinerules/MemoryBank.md` - Clineの記憶管理方法

## cline_docs
- `cline_docs/productContext.md` - プロジェクトの目的と概要
- `cline_docs/activeContext.md` - 現在の作業と次のステップ
- `cline_docs/systemPatterns.md` - システム構造と技術的な決定事項
- `cline_docs/techContext.md` - 使用技術と開発環境
- `cline_docs/progress.md` - 進捗状況

## Build & Test Commands
- Build: `go build`
- Run: `./echonet-list [-debug]`
- Test: `go test ./...`
- Format: `go fmt ./...`
- Check: `go vet ./...`