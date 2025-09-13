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

# root権限チェック
if [[ $EUID -ne 0 ]]; then
    print_error "このスクリプトはroot権限で実行してください: sudo $0"
    exit 1
fi

# サービス存在確認
if ! systemctl list-unit-files | grep -q "^$SERVICE_NAME.service"; then
    print_error "systemdサービス '$SERVICE_NAME' が見つかりません"
    print_info "まず install-systemd.sh でサービスをインストールしてください"
    exit 1
fi

print_info "ECHONET Lite Controller のアップデートを開始します..."

# ファイル存在確認
BINARY_FILE="$PROJECT_DIR/echonet-list"
WEB_BUNDLE_DIR="$PROJECT_DIR/web/bundle"

if [[ ! -f "$BINARY_FILE" ]]; then
    print_error "バイナリファイルが見つかりません: $BINARY_FILE"
    print_info "まず 'go build' でビルドしてください"
    exit 1
fi

if [[ ! -d "$WEB_BUNDLE_DIR" ]]; then
    print_error "Web UIのバンドルが見つかりません: $WEB_BUNDLE_DIR"
    print_info "まず 'cd web && npm run build' でWeb UIをビルドしてください"
    exit 1
fi

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

# バックアップ作成
BACKUP_DIR="/tmp/echonet-backup-$(date +%Y%m%d-%H%M%S)"
print_info "現在のファイルをバックアップしています: $BACKUP_DIR"
mkdir -p "$BACKUP_DIR"

if [[ -f "$INSTALL_DIR/echonet-list" ]]; then
    cp "$INSTALL_DIR/echonet-list" "$BACKUP_DIR/"
fi

if [[ -d "$WEB_DIR" ]]; then
    cp -r "$WEB_DIR" "$BACKUP_DIR/web-bundle" 2>/dev/null || true
fi

# バイナリ更新
print_info "バイナリファイルを更新しています..."
cp "$BINARY_FILE" "$INSTALL_DIR/"
chmod 755 "$INSTALL_DIR/echonet-list"

# Web UI更新
print_info "Web UIファイルを更新しています..."
if [[ -d "$WEB_DIR" ]]; then
    rm -rf "${WEB_DIR:?}"/*
fi
mkdir -p "$WEB_DIR"
cp -r "$WEB_BUNDLE_DIR"/* "$WEB_DIR/"

# 権限再設定
print_info "ファイル権限を設定しています..."
chown -R root:root "$WEB_DIR"

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