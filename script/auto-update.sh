#!/bin/bash

# ECHONET Lite Controller 自動更新スクリプト
# git pull の結果に基づいて自動的にビルドとサービス更新を行う

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 色付きメッセージ関数
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 使用法表示
show_usage() {
    echo "Usage: $0 [-d|--dry-run] [-h|--help]"
    echo ""
    echo "Options:"
    echo "  -d, --dry-run    ドライランモード（実際の操作は行わない）"
    echo "  -h, --help       このヘルプを表示"
    echo ""
    echo "Description:"
    echo "  git pull を実行し、変更されたファイルに応じて自動的にビルドとサービス更新を行います。"
    echo ""
    echo "Actions:"
    echo "  1. git pull で最新のコードを取得"
    echo "  2. 変更ファイルを分析してビルド対象を判定"
    echo "     - Go ファイルの変更 → サーバービルド"
    echo "     - Web UI ファイルの変更 → Web UI ビルド"
    echo "     - 両方の変更 → 全体ビルド"
    echo "  3. 必要に応じて ./script/build.sh を実行"
    echo "  4. ビルドが行われた場合、./script/update.sh でサービス更新"
}

# 引数解析
DRY_RUN=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# スクリプトディレクトリとプロジェクトルートを取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

print_info "ECHONET Lite Controller 自動更新スクリプトを開始します..."
if [[ "$DRY_RUN" == "true" ]]; then
    print_warn "ドライランモード: 実際の操作は行いません"
fi

# Git リポジトリチェック
if [[ ! -d ".git" ]]; then
    print_error "Gitリポジトリが見つかりません"
    exit 1
fi

# Git pull 前の最新コミットハッシュを取得
BEFORE_COMMIT=$(git rev-parse HEAD)

# コミットハッシュの妥当性チェック
if [[ ! "$BEFORE_COMMIT" =~ ^[a-f0-9]{40}$ ]]; then
    print_error "無効なコミットハッシュ形式: $BEFORE_COMMIT"
    exit 1
fi

print_info "現在のコミット: $BEFORE_COMMIT"

# Git pull 実行
if [[ "$DRY_RUN" == "false" ]]; then
    print_info "git pull を実行中..."
    # set -e の影響を避けるため、明示的にエラーハンドリングを行う
    GIT_PULL_OUTPUT=$(git pull 2>&1) || GIT_PULL_EXIT_CODE=$?
    GIT_PULL_EXIT_CODE=${GIT_PULL_EXIT_CODE:-0}

    # git pull の出力を常に表示
    echo "$GIT_PULL_OUTPUT"

    if [[ $GIT_PULL_EXIT_CODE -ne 0 ]]; then
        print_error "git pull が失敗しました (終了コード: $GIT_PULL_EXIT_CODE)"
        print_error "上記の git のエラーメッセージを確認してください"
        print_info "コンフリクトが発生した場合は、手動で解決してから再度実行してください"
        exit 1
    fi

    # "Already up to date" チェック
    if echo "$GIT_PULL_OUTPUT" | grep -q "Already up to date"; then
        print_success "✅ 既に最新です。更新の必要はありません。"
        exit 0
    fi

    # Git pull 後の最新コミットハッシュを取得
    AFTER_COMMIT=$(git rev-parse HEAD)
    print_info "更新後のコミット: $AFTER_COMMIT"
else
    print_info "[DRY-RUN] git pull をスキップ - リモートとの差分をチェック中..."

    # リモートの最新情報を取得（pull せずに fetch のみ）
    git fetch origin >/dev/null 2>&1 || {
        print_error "git fetch が失敗しました"
        exit 1
    }

    # リモートとの差分をチェック（追跡中のブランチを使用）
    REMOTE_COMMIT=$(git rev-parse '@{u}' 2>/dev/null || git rev-parse origin/HEAD 2>/dev/null || git rev-parse origin/main 2>/dev/null || git rev-parse origin/master 2>/dev/null || echo "")

    if [[ -z "$REMOTE_COMMIT" ]]; then
        print_error "リモートブランチが見つかりません"
        exit 1
    fi

    if [[ "$BEFORE_COMMIT" == "$REMOTE_COMMIT" ]]; then
        print_success "✅ 既に最新です。更新の必要はありません。"
        exit 0
    fi

    AFTER_COMMIT="$REMOTE_COMMIT"
    print_info "[DRY-RUN] リモートコミット: $AFTER_COMMIT"
    print_info "[DRY-RUN] 実際には git pull は実行されません"
fi

# 変更されたファイルを取得
print_info "変更されたファイルを分析中..."
CHANGED_FILES=$(git diff --name-only "$BEFORE_COMMIT" "$AFTER_COMMIT" 2>/dev/null || echo "")

if [[ -z "$CHANGED_FILES" ]]; then
    if [[ "$DRY_RUN" == "true" ]]; then
        print_success "✅ リモートとの差分がありません。更新の必要はありません。"
    else
        print_success "✅ 変更されたファイルがありません"
    fi
    exit 0
fi

echo "変更されたファイル:"
while IFS= read -r file; do
    echo "  $file"
done <<< "$CHANGED_FILES"

# 自スクリプトの更新チェック
SCRIPT_NAME="$(basename "$0")"
SCRIPT_RELATIVE_PATH="script/$SCRIPT_NAME"
if echo "$CHANGED_FILES" | grep -q "^$SCRIPT_RELATIVE_PATH$"; then
    print_warn "⚠️  このスクリプト自身が更新されました！"
    print_info "安全のため、更新されたスクリプトで再実行することをお勧めします。"
    print_info ""
    print_info "以下のコマンドで再実行してください:"
    if [[ "$DRY_RUN" == "true" ]]; then
        print_info "  $0 --dry-run"
    else
        print_info "  $0"
    fi
    print_info ""
    read -p "続行しますか？ (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "中断しました。上記のコマンドで再実行してください。"
        exit 0
    fi
fi

# ビルド対象の判定
HAS_GO_CHANGES=false
HAS_WEB_CHANGES=false

if [[ -n "$CHANGED_FILES" ]]; then
    # Go関連ファイルの変更をチェック
    if echo "$CHANGED_FILES" | grep -E '\.go$|^go\.(mod|sum)$' >/dev/null 2>&1; then
        HAS_GO_CHANGES=true
    fi

    # Web UI関連ファイルの変更をチェック（bundle除く）
    if echo "$CHANGED_FILES" | grep -E '^web/' | grep -v -E '^web/bundle/' >/dev/null 2>&1; then
        HAS_WEB_CHANGES=true
    fi
fi

# ビルド戦略の決定
BUILD_TARGET=""
if [[ "$HAS_GO_CHANGES" == "true" && "$HAS_WEB_CHANGES" == "true" ]]; then
    BUILD_TARGET="all"
    print_info "🔧 判定: Go と Web UI の両方に変更があります → 全体ビルド"
elif [[ "$HAS_GO_CHANGES" == "true" ]]; then
    BUILD_TARGET="server"
    print_info "🔧 判定: Go ファイルに変更があります → サーバービルド"
elif [[ "$HAS_WEB_CHANGES" == "true" ]]; then
    BUILD_TARGET="web"
    print_info "🔧 判定: Web UI ファイルに変更があります → Web UI ビルド"
else
    print_info "ℹ️  判定: ビルド対象の変更がありません → ビルドをスキップ"
fi

# ビルド実行
if [[ -n "$BUILD_TARGET" ]]; then
    print_info "🔨 ビルドを開始します: $BUILD_TARGET"

    if [[ "$DRY_RUN" == "false" ]]; then
        if ! "$SCRIPT_DIR/build.sh" "$BUILD_TARGET"; then
            print_error "❌ ビルドが失敗しました"
            exit 1
        fi

        print_success "✅ ビルドが完了しました"
    else
        print_info "[DRY-RUN] ./script/build.sh $BUILD_TARGET をスキップ"
    fi

    # サービス更新の実行
    print_info "🔄 サービス更新を開始します..."

    # update.sh は root 権限が必要なため、sudo で実行
    if [[ "$DRY_RUN" == "false" ]]; then
        if [[ $EUID -ne 0 ]]; then
            print_info "root 権限が必要です。sudo で再実行します..."
            UPDATE_CMD="sudo $SCRIPT_DIR/update.sh"
        else
            UPDATE_CMD="$SCRIPT_DIR/update.sh"
        fi

        if ! $UPDATE_CMD; then
            print_error "❌ サービス更新が失敗しました"
            exit 1
        fi

        print_success "✅ サービス更新が完了しました"
    else
        print_info "[DRY-RUN] ./script/update.sh をスキップ"
    fi

else
    print_success "✅ ビルドが不要のため、更新は完了です"
fi

print_success "🎉 自動更新が正常に完了しました！"

if [[ "$BUILD_TARGET" ]]; then
    print_info ""
    print_info "📊 実行された操作:"
    print_info "  - git pull: 実行済み"
    print_info "  - ビルド: $BUILD_TARGET"
    if [[ "$DRY_RUN" == "false" ]]; then
        print_info "  - サービス更新: 実行済み"
        print_info ""
        # IPv4アドレスを取得（IPv6を避ける）
        HOST_IP=$(hostname -I 2>/dev/null | tr ' ' '\n' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || echo 'localhost')
        print_info "🌐 Web UI: http://${HOST_IP}:8080"
        print_info "🔧 管理: sudo systemctl status echonet-list"
    else
        print_info "  - サービス更新: [DRY-RUN でスキップ]"
    fi
fi