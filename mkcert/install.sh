#!/bin/sh

set -e

# 获取当前脚本所在目录
DIR=$(cd "$(dirname "$0")" >/dev/null 2>&1 && pwd)

# 找到 mkcert 可执行文件路径
MKCERT_BIN=$(command -v mkcert)
if [ -z "$MKCERT_BIN" ]; then
  echo "❌ mkcert 未安装或未在 PATH 中"
  exit 1
fi

# 设置 CAROOT 环境变量为当前目录
export CAROOT="$DIR"

echo "📁 使用 CAROOT 目录：$CAROOT"

# 执行安装，生成根证书（如果不存在）
"$MKCERT_BIN" -install

# 根证书文件路径
ROOT_CA_PEM="$CAROOT/rootCA.pem"
ROOT_CA_KEY="$CAROOT/rootCA-key.pem"

# 检查根证书是否存在
if [ ! -f "$ROOT_CA_PEM" ]; then
  echo "❌ 根证书文件不存在: $ROOT_CA_PEM"
  exit 1
fi

# 重命名备份根证书文件
mv -f "$ROOT_CA_PEM" "$DIR/mycertCA.pem"
cp -f "$DIR/mycertCA.pem" "$DIR/mycertCA.crt"
mv -f "$ROOT_CA_KEY" "$DIR/mycertCA-key.pem"

echo "✅ 根证书生成并重命名为 mycertCA.pem 和 mycertCA-key.pem"
