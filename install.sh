#!/usr/bin/env bash
set -e

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

[[ $EUID -ne 0 ]] && echo -e "${red}错误:${plain} 请使用 root 运行安装脚本" && exit 1

APP_DIR="/usr/local/rx-ui"
SERVICE_NAME="rx-ui"
REPO="${RX_UI_REPO:-DmLeaves/Rx-ui}"
TAG="${RX_UI_TAG:-latest}"
SKIP_SYSTEMD="${RX_UI_SKIP_SYSTEMD:-0}"

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo -e "${red}不支持的架构: ${arch}${plain}"; exit 1 ;;
esac

mkdir -p /usr/local
cd /usr/local

if [[ "$TAG" == "latest" ]]; then
  ASSET_URL="https://github.com/${REPO}/releases/latest/download/rx-ui-linux-${arch}.tar.gz"
  echo -e "${green}下载 Rx-ui latest (${arch})...${plain}"
else
  ASSET_URL="https://github.com/${REPO}/releases/download/${TAG}/rx-ui-linux-${arch}.tar.gz"
  echo -e "${green}下载 Rx-ui ${TAG} (${arch})...${plain}"
fi

wget -O rx-ui-linux-${arch}.tar.gz "${ASSET_URL}"

# 升级时保留数据
if [[ -d "${APP_DIR}" ]]; then
  echo -e "${yellow}检测到旧版本，保留 data/ 并升级...${plain}"
  mkdir -p /tmp/rx-ui-upgrade-backup
  if [[ -d "${APP_DIR}/data" ]]; then
    rm -rf /tmp/rx-ui-upgrade-backup/data
    cp -a "${APP_DIR}/data" /tmp/rx-ui-upgrade-backup/data
  fi
  if [[ "$SKIP_SYSTEMD" != "1" ]]; then
    systemctl stop ${SERVICE_NAME} 2>/dev/null || true
  fi
  rm -rf "${APP_DIR}"
fi

mkdir -p "${APP_DIR}"
tar -xzf rx-ui-linux-${arch}.tar.gz -C "${APP_DIR}"
rm -f rx-ui-linux-${arch}.tar.gz

# 兼容历史/当前不同打包名
if [[ -f "${APP_DIR}/rx-ui" ]]; then
  chmod +x "${APP_DIR}/rx-ui"
elif [[ -f "${APP_DIR}/rx-ui-linux-${arch}" ]]; then
  mv "${APP_DIR}/rx-ui-linux-${arch}" "${APP_DIR}/rx-ui"
  chmod +x "${APP_DIR}/rx-ui"
else
  echo -e "${red}安装失败:${plain} 解压后未找到可执行文件（期望 rx-ui 或 rx-ui-linux-${arch}）"
  exit 1
fi

if [[ -d /tmp/rx-ui-upgrade-backup/data ]]; then
  mkdir -p "${APP_DIR}/data"
  cp -a /tmp/rx-ui-upgrade-backup/data/. "${APP_DIR}/data/"
fi

wget -O /etc/systemd/system/${SERVICE_NAME}.service "https://raw.githubusercontent.com/${REPO}/main/rx-ui.service"
wget -O /usr/bin/Rx-ui "https://raw.githubusercontent.com/${REPO}/main/Rx-ui.sh"
chmod +x /usr/bin/Rx-ui
ln -sf /usr/bin/Rx-ui /usr/bin/rx-ui

if [[ "$SKIP_SYSTEMD" == "1" ]]; then
  echo -e "${yellow}跳过 systemd 操作（RX_UI_SKIP_SYSTEMD=1）${plain}"
else
  systemctl daemon-reload
  systemctl enable ${SERVICE_NAME}
  systemctl restart ${SERVICE_NAME}
fi

echo -e "${green}安装完成${plain}"
echo "------------------------------------------"
echo "命令行菜单: Rx-ui"
echo "服务名称: ${SERVICE_NAME}"
echo "常用命令:"
echo "  Rx-ui status"
echo "  Rx-ui set-port"
echo "  Rx-ui reset-admin"
echo "  Rx-ui update"
echo "------------------------------------------"
