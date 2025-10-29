#!/bin/bash

# ECHONET Lite Controller アップデートスクリプト
# カレントディレクトリのビルド済みバイナリとWeb UIでsystemdサービスを更新

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 設定
SERVICE_NAME="echonet-list"
SERVICE_USER="echonet"
SERVICE_GROUP="echonet"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/echonet-list"
WEB_DIR="/usr/local/share/echonet-list/web"

# オプション
WEB_ONLY=false

# 色付きメッセージ
print_info() {
    echo -e "\033[32m[INFO]\033[0m $1"
}

print_warn() {
    echo -e "\033[33m[WARN]\033[0m $1"
}

print_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

print_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

# 使用法表示
show_usage() {
    echo "Usage: $0 [--web-only] [-h|--help]"
    echo ""
    echo "Options:"
    echo "  --web-only       Web UIファイルのみ更新（サービス再起動なし）"
    echo "  -h, --help       このヘルプを表示"
    echo ""
    echo "Description:"
    echo "  systemdサービスとしてインストールされたECHONET Lite Controllerを更新します。"
    echo ""
    echo "Modes:"
    echo "  通常モード:      バイナリとWeb UIを更新し、サービスを再起動"
    echo "  --web-only:      Web UIファイルのみコピー（サービス再起動なし、ダウンタイムなし）"
}

# 引数解析
while [[ $# -gt 0 ]]; do
    case $1 in
        --web-only)
            WEB_ONLY=true
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

# root権限チェック
if [[ $EUID -ne 0 ]]; then
    print_error "このスクリプトはroot権限で実行してください: sudo $0"
    exit 1
fi

# Web-onlyモード以外ではサービス存在確認
if [[ "$WEB_ONLY" == "false" ]]; then
    if ! systemctl list-unit-files | grep -q "^$SERVICE_NAME.service"; then
        print_error "systemdサービス '$SERVICE_NAME' が見つかりません"
        print_info "まず install-systemd.sh でサービスをインストールしてください"
        exit 1
    fi
fi

if [[ "$WEB_ONLY" == "true" ]]; then
    print_info "Web UI ファイルのみを更新します（サービス再起動なし）..."
else
    print_info "ECHONET Lite Controller のアップデートを開始します..."
fi

# ファイル存在確認
BINARY_FILE="$PROJECT_DIR/echonet-list"
WEB_BUNDLE_DIR="$PROJECT_DIR/web/bundle"

if [[ "$WEB_ONLY" == "false" && ! -f "$BINARY_FILE" ]]; then
    print_error "バイナリファイルが見つかりません: $BINARY_FILE"
    print_info "まず 'go build' でビルドしてください"
    exit 1
fi

if [[ ! -d "$WEB_BUNDLE_DIR" ]]; then
    print_error "Web UIのバンドルが見つかりません: $WEB_BUNDLE_DIR"
    print_info "まず 'cd web && npm run build' でWeb UIをビルドしてください"
    exit 1
fi

# バックアップ作成（全モード共通）
BACKUP_DIR="/tmp/echonet-backup-$(date +%Y%m%d-%H%M%S)"
print_info "現在のファイルをバックアップしています: $BACKUP_DIR"
mkdir -p "$BACKUP_DIR"

# サーバーバイナリのバックアップ（参照用、整合性確保のため）
if [[ -f "$INSTALL_DIR/echonet-list" ]]; then
    cp "$INSTALL_DIR/echonet-list" "$BACKUP_DIR/"
fi

# Web UIファイルのバックアップ
if [[ -d "$WEB_DIR" ]]; then
    cp -r "$WEB_DIR" "$BACKUP_DIR/web-bundle" 2>/dev/null || true
fi

# Web-onlyモード以外では、バージョン情報表示、サービス停止、バイナリ更新を実行
if [[ "$WEB_ONLY" == "false" ]]; then
    # バージョン情報表示（可能な場合）
    print_info "現在のバージョン情報:"
    if [[ -f "$INSTALL_DIR/echonet-list" ]]; then
        CURRENT_VERSION=$("$INSTALL_DIR/echonet-list" -version 2>/dev/null || echo "不明")
        print_info "  インストール済み: $CURRENT_VERSION"
    fi

    NEW_VERSION=$("$BINARY_FILE" -version 2>/dev/null || echo "不明")
    print_info "  新しいバージョン: $NEW_VERSION"

    # サービス停止
    WAS_ACTIVE=false
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        print_info "サービスを停止しています..."
        systemctl stop "$SERVICE_NAME"
        WAS_ACTIVE=true
    else
        print_info "サービスは停止状態です"
    fi

    # バイナリ更新
    print_info "バイナリファイルを更新しています..."
    cp "$BINARY_FILE" "$INSTALL_DIR/"
    chmod 755 "$INSTALL_DIR/echonet-list"
fi

# Web UI更新（アトミックな置換でレースコンディションを回避）
print_info "Web UIファイルを更新しています..."
WEB_DIR_NEW="${WEB_DIR}.new"
WEB_DIR_OLD="${WEB_DIR}.old"

# クリーンアップ（以前の失敗時の残骸があれば削除）
rm -rf "$WEB_DIR_NEW" "$WEB_DIR_OLD" 2>/dev/null || true

# 新しいディレクトリにファイルをコピー
mkdir -p "$WEB_DIR_NEW"
cp -r "$WEB_BUNDLE_DIR"/* "$WEB_DIR_NEW/"

# 権限設定（移動前に実行）
chown -R root:root "$WEB_DIR_NEW"

# アトミックな置換: 既存 → .old、新規 → 正式な場所
if [[ -d "$WEB_DIR" ]]; then
    mv "$WEB_DIR" "$WEB_DIR_OLD" 2>/dev/null || true
fi
mv "$WEB_DIR_NEW" "$WEB_DIR"

# 古いディレクトリを削除
rm -rf "$WEB_DIR_OLD" 2>/dev/null || true

# Web-onlyモード以外では、その他の更新とサービス開始を実行
if [[ "$WEB_ONLY" == "false" ]]; then
    # デバイス情報ファイルの更新（新しいファイルがある場合）
    for file in devices.json groups.json aliases.json; do
        if [[ -f "$PROJECT_DIR/$file" ]]; then
            # タイムスタンプ比較
            if [[ ! -f "$DATA_DIR/$file" ]] || [[ "$PROJECT_DIR/$file" -nt "$DATA_DIR/$file" ]]; then
                print_info "$file を更新しています..."
                cp "$PROJECT_DIR/$file" "$DATA_DIR/"
                chown "$SERVICE_USER:$SERVICE_GROUP" "$DATA_DIR/$file"
            fi
        fi
    done

    # systemdサービスファイルの更新チェック
    if [[ -f "$PROJECT_DIR/systemd/echonet-list.service" ]]; then
        if [[ ! -f "/etc/systemd/system/echonet-list.service" ]] ||
           [[ "$PROJECT_DIR/systemd/echonet-list.service" -nt "/etc/systemd/system/echonet-list.service" ]]; then
            print_info "systemdサービスファイルを更新しています..."
            cp "$PROJECT_DIR/systemd/echonet-list.service" "/etc/systemd/system/"
            systemctl daemon-reload
        fi
    fi

    # logrotate設定の更新チェック
    if [[ -f "$PROJECT_DIR/systemd/echonet-list.logrotate" ]]; then
        if [[ ! -f "/etc/logrotate.d/echonet-list" ]] ||
           [[ "$PROJECT_DIR/systemd/echonet-list.logrotate" -nt "/etc/logrotate.d/echonet-list" ]]; then
            print_info "logrotate設定を更新しています..."
            cp "$PROJECT_DIR/systemd/echonet-list.logrotate" "/etc/logrotate.d/echonet-list"
        fi
    fi

    # サービス開始（元々起動していた場合）
    if [[ "$WAS_ACTIVE" == "true" ]]; then
        print_info "サービスを開始しています..."
        systemctl start "$SERVICE_NAME"

        # 起動確認
        sleep 2
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            print_success "✅ サービスが正常に開始されました"
        else
            print_error "❌ サービスの開始に失敗しました"
            print_info "バックアップからロールバックしています..."

            # ロールバック
            if [[ -f "$BACKUP_DIR/echonet-list" ]]; then
                cp "$BACKUP_DIR/echonet-list" "$INSTALL_DIR/"
            fi
            if [[ -d "$BACKUP_DIR/web-bundle" ]]; then
                rm -rf "${WEB_DIR:?}"/*
                cp -r "$BACKUP_DIR/web-bundle"/* "$WEB_DIR/"
            fi

            systemctl start "$SERVICE_NAME"
            print_error "ロールバックしました。詳細は以下のコマンドで確認してください:"
            print_error "  sudo systemctl status $SERVICE_NAME"
            print_error "  sudo journalctl -u $SERVICE_NAME"
            exit 1
        fi
    else
        print_info "サービスは停止状態のままです"
    fi

    print_success "✅ アップデートが完了しました！"
    print_info ""
    print_info "📊 更新情報:"
    if [[ "$CURRENT_VERSION" != "$NEW_VERSION" ]]; then
        print_info "  バージョン: $CURRENT_VERSION → $NEW_VERSION"
    fi
    print_info "  バックアップ: $BACKUP_DIR"
    print_info ""
    print_info "🔧 管理コマンド:"
    print_info "  サービス状態確認: sudo systemctl status $SERVICE_NAME"
    print_info "  ログ確認:         sudo journalctl -u $SERVICE_NAME -f"
    print_info "  Web UI:          http://$(hostname -I | awk '{print $1}'):8080"

    # バックアップの自動削除確認
    print_info ""
    print_info "💾 バックアップファイルは自動削除されません"
    print_info "   不要になったら手動で削除してください: rm -rf $BACKUP_DIR"
else
    # Web-onlyモードの完了メッセージ
    print_success "✅ Web UI ファイルの更新が完了しました！"
    print_info ""
    print_info "📊 更新情報:"
    print_info "  更新内容: Web UIファイルのみ"
    print_info "  配置先: $WEB_DIR"
    print_info "  バックアップ: $BACKUP_DIR"
    print_info "  サービス状態: 再起動なし（ダウンタイムなし）"
    print_info ""
    print_info "ℹ️  サーバーは動的にファイルを読み込むため、すぐに反映されます"
    print_info "  Web UI:          http://$(hostname -I | awk '{print $1}'):8080"
    print_info ""
    print_info "🔄 ロールバック方法（問題があった場合）:"
    print_info "  sudo rm -rf $WEB_DIR/*"
    print_info "  sudo cp -r $BACKUP_DIR/web-bundle/* $WEB_DIR/"
    print_info ""
    print_info "💾 バックアップファイルは自動削除されません"
    print_info "   不要になったら手動で削除してください: rm -rf $BACKUP_DIR"
fi