#!/bin/bash

# ECHONET Lite Controller systemd アンインストールスクリプト
# Raspberry Pi (Ubuntu) 環境用

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 設定
SERVICE_NAME="echonet-list"
SERVICE_USER="echonet"
SERVICE_GROUP="echonet"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/echonet-list"
DATA_DIR="/var/lib/echonet-list"
LOG_DIR="/var/log"
WEB_DIR="/usr/local/share/echonet-list"

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

print_question() {
    echo -e "\033[36m[QUESTION]\033[0m $1"
}

# root権限チェック
if [[ $EUID -ne 0 ]]; then
    print_error "このスクリプトはroot権限で実行してください: sudo $0"
    exit 1
fi

print_info "ECHONET Lite Controller のアンインストールを開始します..."

# 確認プロンプト
read -p "$(print_question "本当にアンインストールしますか？ (y/N): ")" -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "アンインストールをキャンセルしました"
    exit 0
fi

# データ保持の確認
KEEP_DATA=false
if [[ -d "$DATA_DIR" ]]; then
    read -p "$(print_question "データディレクトリ ($DATA_DIR) の内容を保持しますか？ (Y/n): ")" -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        KEEP_DATA=true
        print_info "データディレクトリは保持されます"
    fi
fi

# 設定ファイル保持の確認
KEEP_CONFIG=false
if [[ -d "$CONFIG_DIR" ]]; then
    read -p "$(print_question "設定ディレクトリ ($CONFIG_DIR) の内容を保持しますか？ (Y/n): ")" -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        KEEP_CONFIG=true
        print_info "設定ディレクトリは保持されます"
    fi
fi

# サービス停止
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    print_info "サービスを停止しています..."
    systemctl stop "$SERVICE_NAME" || true
fi

# サービス無効化
if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
    print_info "サービスを無効化しています..."
    systemctl disable "$SERVICE_NAME" || true
fi

# systemdサービスファイル削除
if [[ -f "/etc/systemd/system/$SERVICE_NAME.service" ]]; then
    print_info "systemdサービスファイルを削除しています..."
    rm -f "/etc/systemd/system/$SERVICE_NAME.service"
fi

# systemd設定の再読み込み
print_info "systemd設定を再読み込みしています..."
systemctl daemon-reload

# バイナリファイル削除
if [[ -f "$INSTALL_DIR/echonet-list" ]]; then
    print_info "バイナリファイルを削除しています..."
    rm -f "$INSTALL_DIR/echonet-list"
fi

# Web UIファイル削除
if [[ -d "$WEB_DIR" ]]; then
    print_info "Web UIファイルを削除しています..."
    rm -rf "$WEB_DIR"
fi

# logrotate設定削除
if [[ -f "/etc/logrotate.d/echonet-list" ]]; then
    print_info "logrotate設定を削除しています..."
    rm -f "/etc/logrotate.d/echonet-list"
fi

# ログファイル削除
if [[ -f "$LOG_DIR/echonet-list.log" ]]; then
    print_info "ログファイルを削除しています..."
    rm -f "$LOG_DIR/echonet-list.log"*
fi

# PIDファイル削除
if [[ -f "/var/run/echonet-list.pid" ]]; then
    print_info "PIDファイルを削除しています..."
    rm -f "/var/run/echonet-list.pid"
fi

# tmpfiles設定削除
if [[ -f "/etc/tmpfiles.d/echonet-list.conf" ]]; then
    print_info "tmpfiles設定を削除しています..."
    rm -f "/etc/tmpfiles.d/echonet-list.conf"
fi

# ランタイムディレクトリ削除
if [[ -d "/run/echonet-list" ]]; then
    print_info "ランタイムディレクトリを削除しています..."
    rm -rf "/run/echonet-list"
fi

# データディレクトリ処理
if [[ -d "$DATA_DIR" ]]; then
    if [[ "$KEEP_DATA" == "true" ]]; then
        print_info "データディレクトリを保持しています: $DATA_DIR"
    else
        print_info "データディレクトリを削除しています..."
        rm -rf "$DATA_DIR"
    fi
fi

# 設定ディレクトリ処理
if [[ -d "$CONFIG_DIR" ]]; then
    if [[ "$KEEP_CONFIG" == "true" ]]; then
        print_info "設定ディレクトリを保持しています: $CONFIG_DIR"
        print_info "  (証明書、設定ファイル含む)"
    else
        print_info "設定ディレクトリを削除しています..."
        print_info "  (証明書、設定ファイル含む)"
        rm -rf "$CONFIG_DIR"
    fi
fi

# ユーザー・グループ削除
if id "$SERVICE_USER" &>/dev/null; then
    print_info "サービス用ユーザー '$SERVICE_USER' を削除しています..."
    userdel "$SERVICE_USER" || true
fi

if getent group "$SERVICE_GROUP" >/dev/null 2>&1; then
    print_info "サービス用グループ '$SERVICE_GROUP' を削除しています..."
    groupdel "$SERVICE_GROUP" || true
fi

print_info "✅ アンインストールが完了しました！"

# 保持されたファイルの確認
if [[ "$KEEP_DATA" == "true" || "$KEEP_CONFIG" == "true" ]]; then
    print_info ""
    print_info "🗂️  保持されたファイル:"
    if [[ "$KEEP_DATA" == "true" ]]; then
        print_info "  データ: $DATA_DIR"
    fi
    if [[ "$KEEP_CONFIG" == "true" ]]; then
        print_info "  設定: $CONFIG_DIR"
    fi
    print_warn "これらのディレクトリは手動で削除してください（必要に応じて）"
fi

print_info ""
print_info "🔄 再インストールするには:"
print_info "  $SCRIPT_DIR/install-systemd.sh を実行してください"