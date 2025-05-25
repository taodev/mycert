#!/bin/sh

set -e

# GitHub 证书下载地址（raw 内容）
REMOTE_CERT_URL="https://github.com/taodev/mycert/raw/main/mycert.pem"
CERT_FILE="mycertCA.pem"
CERT_NAME="mycertCA"
CRT_PATH=""
OS_ID=$(grep ^ID= /etc/os-release | cut -d= -f2 | tr -d '"')

# 检查命令是否存在
install_if_missing() {
  CMD=$1
  PKG=$2

  if ! command -v "$CMD" >/dev/null 2>&1; then
    echo "🔍 缺失命令 $CMD，安装 $PKG..."
    case "$OS_ID" in
      ubuntu|debian)
        apt update && apt install -y "$PKG"
        ;;
      centos|rhel)
        yum install -y "$PKG"
        ;;
      rocky|almalinux)
        dnf install -y "$PKG"
        ;;
      alpine)
        apk update && apk add "$PKG"
        ;;
      arch)
        pacman -Sy --noconfirm "$PKG"
        ;;
      *)
        echo "❌ 不支持的发行版: $OS_ID"
        exit 1
        ;;
    esac
  fi
}

# 下载证书（如果本地不存在）
download_cert() {
  if [ ! -f "$CERT_FILE" ]; then
    echo "🌐 未检测到本地证书，尝试从 GitHub 下载..."
    install_if_missing curl curl
    curl -fsSL "$REMOTE_CERT_URL" -o "$CERT_FILE"
    echo "✅ 证书下载成功: $CERT_FILE"
  else
    echo "📁 本地已存在证书: $CERT_FILE"
  fi
}

# 安装证书
install_cert() {
  download_cert
  echo "📦 安装根证书到 $OS_ID ..."

  case "$OS_ID" in
    ubuntu|debian)
      install_if_missing update-ca-certificates ca-certificates
      CRT_PATH="/usr/local/share/ca-certificates/${CERT_NAME}.crt"
      cp "$CERT_FILE" "$CRT_PATH"
      update-ca-certificates
      ;;

    centos|rhel|rocky|almalinux)
      install_if_missing update-ca-trust ca-certificates
      CRT_PATH="/etc/pki/ca-trust/source/anchors/${CERT_NAME}.pem"
      cp "$CERT_FILE" "$CRT_PATH"
      update-ca-trust extract
      ;;

    alpine)
      install_if_missing update-ca-certificates ca-certificates
      CRT_PATH="/usr/local/share/ca-certificates/${CERT_NAME}.crt"
      cp "$CERT_FILE" "$CRT_PATH"
      update-ca-certificates
      ;;

    arch)
      install_if_missing trust ca-certificates-utils
      CRT_PATH="/etc/ca-certificates/trust-source/anchors/${CERT_NAME}.crt"
      cp "$CERT_FILE" "$CRT_PATH"
      trust extract-compat
      ;;

    *)
      echo "❌ 不支持的系统: $OS_ID"
      exit 1
      ;;
  esac

  echo "✅ 证书安装成功: $CRT_PATH"
}

# 卸载证书
uninstall_cert() {
  echo "🗑️ 卸载根证书..."
  case "$OS_ID" in
    ubuntu|debian|alpine)
      CRT_PATH="/usr/local/share/ca-certificates/${CERT_NAME}.crt"
      rm -f "$CRT_PATH"
      update-ca-certificates --fresh
      ;;

    centos|rhel|rocky|almalinux)
      CRT_PATH="/etc/pki/ca-trust/source/anchors/${CERT_NAME}.pem"
      rm -f "$CRT_PATH"
      update-ca-trust extract
      ;;

    arch)
      CRT_PATH="/etc/ca-certificates/trust-source/anchors/${CERT_NAME}.crt"
      rm -f "$CRT_PATH"
      trust extract-compat
      ;;

    *)
      echo "❌ 不支持的系统: $OS_ID"
      exit 1
      ;;
  esac
  echo "✅ 证书已卸载: $CRT_PATH"
}

# 检查证书是否已安装
check_cert() {
  echo "🔍 检查证书是否已安装..."
  case "$OS_ID" in
    ubuntu|debian|alpine)
      ls "/usr/local/share/ca-certificates/${CERT_NAME}.crt" >/dev/null 2>&1 && echo "✅ 已安装" || echo "❌ 未安装"
      ;;

    centos|rhel|rocky|almalinux)
      ls "/etc/pki/ca-trust/source/anchors/${CERT_NAME}.pem" >/dev/null 2>&1 && echo "✅ 已安装" || echo "❌ 未安装"
      ;;

    arch)
      ls "/etc/ca-certificates/trust-source/anchors/${CERT_NAME}.crt" >/dev/null 2>&1 && echo "✅ 已安装" || echo "❌ 未安装"
      ;;

    *)
      echo "❌ 不支持的系统: $OS_ID"
      ;;
  esac
}

# 主程序入口
case "$1" in
  install)
    install_cert
    ;;
  uninstall)
    uninstall_cert
    ;;
  check)
    check_cert
    ;;
  *)
    echo "用法: $0 {install|uninstall|check}"
    ;;
esac
