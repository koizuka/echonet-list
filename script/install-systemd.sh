#!/bin/bash

# ECHONET Lite Controller systemd セットアップスクリプト
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

# root権限チェック
if [[ $EUID -ne 0 ]]; then
    print_error "このスクリプトはroot権限で実行してください: sudo $0"
    exit 1
fi

# 必要なファイルの存在確認
if [[ ! -f "$PROJECT_DIR/echonet-list" ]]; then
    print_error "バイナリファイルが見つかりません: $PROJECT_DIR/echonet-list"
    print_info "まず 'go build' でビルドしてください"
    exit 1
fi

if [[ ! -d "$PROJECT_DIR/web/bundle" ]]; then
    print_error "Web UIのバンドルが見つかりません: $PROJECT_DIR/web/bundle"
    print_info "まず 'cd web && npm run build' でWeb UIをビルドしてください"
    exit 1
fi

print_info "ECHONET Lite Controller のsystemdセットアップを開始します..."

# サービス停止（既に存在する場合）
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    print_info "既存のサービスを停止しています..."
    systemctl stop "$SERVICE_NAME"
fi

# ユーザー・グループ作成
if ! getent group "$SERVICE_GROUP" >/dev/null; then
    print_info "サービス用グループ '$SERVICE_GROUP' を作成しています..."
    groupadd "$SERVICE_GROUP"
fi

if ! id "$SERVICE_USER" &>/dev/null; then
    print_info "サービス用ユーザー '$SERVICE_USER' を作成しています..."
    useradd --system --gid "$SERVICE_GROUP" --home-dir "$DATA_DIR" --shell /sbin/nologin "$SERVICE_USER"
else
    print_info "サービス用ユーザー '$SERVICE_USER' は既に存在します"
fi

# ディレクトリ作成
print_info "必要なディレクトリを作成しています..."
mkdir -p "$CONFIG_DIR"
mkdir -p "$CONFIG_DIR/certs"
mkdir -p "$DATA_DIR"
mkdir -p "$WEB_DIR"

# バイナリのインストール
print_info "バイナリファイルをインストールしています..."
cp "$PROJECT_DIR/echonet-list" "$INSTALL_DIR/"
chmod 755 "$INSTALL_DIR/echonet-list"

# Web UIのインストール
print_info "Web UIファイルをインストールしています..."
cp -r "$PROJECT_DIR/web/bundle"/* "$WEB_DIR/"

# TLS証明書のインストール
if [[ -d "$PROJECT_DIR/certs" ]]; then
    print_info "TLS証明書をインストールしています..."
    cp "$PROJECT_DIR/certs"/* "$CONFIG_DIR/certs/"
else
    print_warn "証明書ディレクトリが見つかりません: $PROJECT_DIR/certs"
    print_info "TLSを使用する場合は、事前に証明書を準備してください"
fi

# 設定ファイルのインストール
if [[ ! -f "$CONFIG_DIR/config.toml" ]]; then
    print_info "設定ファイルをインストールしています..."
    cp "$PROJECT_DIR/systemd/config.toml.systemd" "$CONFIG_DIR/config.toml"
else
    print_warn "設定ファイル $CONFIG_DIR/config.toml は既に存在するため、スキップします"
fi

# デバイス情報ファイルのコピー（存在する場合）
for file in devices.json groups.json aliases.json; do
    if [[ -f "$PROJECT_DIR/$file" ]]; then
        print_info "$file をコピーしています..."
        cp "$PROJECT_DIR/$file" "$DATA_DIR/"
    fi
done

# logrotate設定のインストール
if [[ -f "$PROJECT_DIR/systemd/echonet-list.logrotate" ]]; then
    print_info "logrotate設定をインストールしています..."
    cp "$PROJECT_DIR/systemd/echonet-list.logrotate" "/etc/logrotate.d/echonet-list"
fi

# systemd tmpfiles設定（PIDファイル用ディレクトリ）
print_info "systemd tmpfiles設定を作成しています..."
cat > /etc/tmpfiles.d/echonet-list.conf << 'EOF'
# ECHONET Lite Controller runtime directory
d /run/echonet-list 0755 echonet echonet -
f /run/echonet-list/echonet-list.pid 0644 echonet echonet -
EOF

# tmpfiles設定を適用
print_info "tmpfiles設定を適用しています..."
systemd-tmpfiles --create /etc/tmpfiles.d/echonet-list.conf

# 権限設定
print_info "ファイル権限を設定しています..."
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$DATA_DIR"
chown -R root:root "$WEB_DIR"
chown root:"$SERVICE_GROUP" "$CONFIG_DIR/config.toml"
chown -R root:"$SERVICE_GROUP" "$CONFIG_DIR/certs"
chmod 640 "$CONFIG_DIR/config.toml"
chmod 750 "$DATA_DIR"
chmod 750 "$CONFIG_DIR/certs"
chmod 640 "$CONFIG_DIR/certs"/*

# ログファイルの権限設定
touch "$LOG_DIR/echonet-list.log"
chown "$SERVICE_USER:$SERVICE_GROUP" "$LOG_DIR/echonet-list.log"
chmod 644 "$LOG_DIR/echonet-list.log"

# systemdサービスファイルのインストール
print_info "systemdサービスファイルをインストールしています..."
cp "$PROJECT_DIR/systemd/echonet-list.service" "/etc/systemd/system/"

# systemd設定の再読み込み
print_info "systemd設定を再読み込みしています..."
systemctl daemon-reload

# サービスの有効化
print_info "サービスを有効化しています..."
systemctl enable "$SERVICE_NAME"

# サービスの開始
print_info "サービスを開始しています..."
systemctl start "$SERVICE_NAME"

# インストール完了の確認
sleep 2
if systemctl is-active --quiet "$SERVICE_NAME"; then
    print_info "✅ インストールが完了しました！"
    print_info ""
    print_info "🔧 管理コマンド:"
    print_info "  サービス状態確認: sudo systemctl status $SERVICE_NAME"
    print_info "  サービス停止:     sudo systemctl stop $SERVICE_NAME"
    print_info "  サービス開始:     sudo systemctl start $SERVICE_NAME"
    print_info "  サービス再起動:   sudo systemctl restart $SERVICE_NAME"
    print_info "  ログ確認:         sudo journalctl -u $SERVICE_NAME -f"
    print_info "  ログファイル:     $LOG_DIR/echonet-list.log"
    print_info ""
    print_info "🌐 Web UI: https://$(hostname -I | awk '{print $1}'):8080"
    print_info "📁 設定ファイル: $CONFIG_DIR/config.toml"
    print_info "📁 データディレクトリ: $DATA_DIR"
else
    print_error "❌ サービスの開始に失敗しました"
    print_info "詳細は以下のコマンドで確認してください:"
    print_info "  sudo systemctl status $SERVICE_NAME"
    print_info "  sudo journalctl -u $SERVICE_NAME"
    exit 1
fi