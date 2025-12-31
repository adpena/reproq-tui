#!/usr/bin/env bash
set -euo pipefail

REPO="adpena/reproq-tui"
BIN_NAME="reproq-tui"
VERSION="${VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-}"

if [[ -z "${INSTALL_DIR}" ]]; then
  if [[ -w "/usr/local/bin" ]]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
  Darwin) OS="darwin" ;;
  Linux) OS="linux" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  *)
    echo "Unsupported OS: ${OS}" >&2
    exit 1
    ;;
esac

case "${ARCH}" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

if [[ -z "${VERSION}" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p' | head -n1)"
fi

if [[ -z "${VERSION}" ]]; then
  echo "Unable to determine latest version. Set VERSION=0.0.101 and retry." >&2
  exit 1
fi

ARCHIVE="${BIN_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "Downloading ${URL}..."
curl -fsSL "${URL}" -o "${TMP_DIR}/${ARCHIVE}"

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"

mkdir -p "${INSTALL_DIR}"

TARGET="${INSTALL_DIR}/${BIN_NAME}"
SUDO=""
if [[ ! -w "${INSTALL_DIR}" ]]; then
  if command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
  else
    echo "Install dir not writable and sudo not available: ${INSTALL_DIR}" >&2
    exit 1
  fi
fi

if command -v install >/dev/null 2>&1; then
  ${SUDO} install -m 0755 "${TMP_DIR}/${BIN_NAME}" "${TARGET}"
else
  ${SUDO} cp "${TMP_DIR}/${BIN_NAME}" "${TARGET}"
  ${SUDO} chmod 0755 "${TARGET}"
fi

echo "Installed ${BIN_NAME} to ${TARGET}"
if ! command -v "${BIN_NAME}" >/dev/null 2>&1; then
  echo "Ensure ${INSTALL_DIR} is in your PATH."
fi
