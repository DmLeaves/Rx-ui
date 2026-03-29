#!/usr/bin/env bash

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

APP_NAME="rx-ui"
SCRIPT_VERSION="v1.1.4"
APP_DIR="/usr/local/rx-ui"
APP_BIN="${APP_DIR}/rx-ui"
SERVICE_NAME="rx-ui"

[[ $EUID -ne 0 ]] && echo -e "${red}错误:${plain} 请使用 root 运行" && exit 1

logi(){ echo -e "${green}[INF]${plain} $*"; }
loge(){ echo -e "${red}[ERR]${plain} $*"; }

check_install() {
  [[ -x "$APP_BIN" ]] || { loge "未检测到 ${APP_BIN}，请先安装"; return 1; }
  return 0
}

run_app() {
  (cd "$APP_DIR" && "$APP_BIN" "$@")
}

get_installed_version() {
  [[ -f "${APP_DIR}/VERSION" ]] && cat "${APP_DIR}/VERSION" || echo "unknown"
}

start_panel(){ systemctl start "$SERVICE_NAME"; logi "已启动"; }
stop_panel(){ systemctl stop "$SERVICE_NAME"; logi "已停止"; }
restart_panel(){ systemctl restart "$SERVICE_NAME"; logi "已重启"; }
status_panel(){ systemctl status "$SERVICE_NAME" --no-pager -l; }
logs_panel(){ journalctl -u "$SERVICE_NAME" -e --no-pager -n 100; }

enable_panel(){ systemctl enable "$SERVICE_NAME"; }
disable_panel(){ systemctl disable "$SERVICE_NAME"; }

set_port(){
  read -rp "输入新端口 [1-65535]: " port
  [[ -z "$port" ]] && return
  if ! [[ "$port" =~ ^[0-9]+$ ]] || ((port < 1 || port > 65535)); then
    loge "端口必须在 1-65535"
    return
  fi
  run_app setting -port "$port"
  logi "端口已更新，建议重启服务"
}

reset_admin(){
  read -rp "新用户名 [默认 admin]: " user
  read -rp "新密码 [默认 admin123]: " pass
  user=${user:-admin}
  pass=${pass:-admin123}
  run_app setting -username "$user" -password "$pass"
  logi "管理员账号已更新"
}

show_setting(){ run_app setting -show; }

update_panel(){
  read -rp "确认升级到最新版本? [y/N]: " c
  [[ "$c" =~ ^[Yy]$ ]] || return
  bash <(curl -Ls https://raw.githubusercontent.com/DmLeaves/Rx-ui/main/install.sh)
}

uninstall_panel(){
  read -rp "确认卸载 Rx-ui? [y/N]: " c
  [[ "$c" =~ ^[Yy]$ ]] || return
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  systemctl disable "$SERVICE_NAME" 2>/dev/null || true
  rm -f /etc/systemd/system/${SERVICE_NAME}.service
  systemctl daemon-reload
  rm -rf "$APP_DIR"
  rm -f /usr/bin/Rx-ui /usr/bin/rx-ui
  logi "已卸载"
}

show_usage(){
  cat <<EOF
Rx-ui 管理命令 (script: ${SCRIPT_VERSION}, installed: $(get_installed_version)):
  Rx-ui                打开交互菜单
  Rx-ui start          启动
  Rx-ui stop           停止
  Rx-ui restart        重启
  Rx-ui status         状态
  Rx-ui log            日志
  Rx-ui enable         开机自启
  Rx-ui disable        取消开机自启
  Rx-ui setting        查看设置
  Rx-ui set-port       修改面板端口
  Rx-ui reset-admin    重置管理员账号密码
  Rx-ui update         升级
  Rx-ui uninstall      卸载
EOF
}

show_menu(){
  while true; do
    echo -e "\n${green}Rx-ui 管理菜单${plain} (script: ${SCRIPT_VERSION} | installed: $(get_installed_version))"
    echo " 0. 退出"
    echo " 1. 启动"
    echo " 2. 停止"
    echo " 3. 重启"
    echo " 4. 状态"
    echo " 5. 日志"
    echo " 6. 开机自启"
    echo " 7. 取消开机自启"
    echo " 8. 修改面板端口"
    echo " 9. 重置管理员账号/密码"
    echo "10. 查看当前设置"
    echo "11. 升级"
    echo "12. 卸载"
    read -rp "请选择 [0-12]: " n
    case "$n" in
      0) exit 0 ;;
      1) check_install && start_panel ;;
      2) check_install && stop_panel ;;
      3) check_install && restart_panel ;;
      4) check_install && status_panel ;;
      5) check_install && logs_panel ;;
      6) check_install && enable_panel ;;
      7) check_install && disable_panel ;;
      8) check_install && set_port ;;
      9) check_install && reset_admin ;;
      10) check_install && show_setting ;;
      11) update_panel ; exit 0 ;;
      12) uninstall_panel ; exit 0 ;;
      *) loge "请输入 0-12" ;;
    esac
  done
}

case "${1:-}" in
  "") show_menu ;;
  start) check_install && start_panel ;;
  stop) check_install && stop_panel ;;
  restart) check_install && restart_panel ;;
  status) check_install && status_panel ;;
  log) check_install && logs_panel ;;
  enable) check_install && enable_panel ;;
  disable) check_install && disable_panel ;;
  setting) check_install && show_setting ;;
  set-port) check_install && set_port ;;
  reset-admin) check_install && reset_admin ;;
  update) update_panel ;;
  uninstall) uninstall_panel ;;
  *) show_usage ;;
esac
