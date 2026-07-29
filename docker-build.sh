#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="${IMAGE_NAME:-mt1976/frantic-postr}"
VERSION_FILE="${VERSION_FILE:-version.no}"
CREDENTIALS_FILE="${CREDENTIALS_FILE:-.dockerhub.env}"
DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:-}"
DOCKERHUB_PASSWORD="${DOCKERHUB_PASSWORD:-}"
DOCKERHUB_TOKEN="${DOCKERHUB_TOKEN:-}"
DRY_RUN="${DRY_RUN:-0}"

if [ -f "$CREDENTIALS_FILE" ]; then
  # shellcheck disable=SC1090
  source "$CREDENTIALS_FILE"
fi

if [ -z "$DOCKERHUB_USERNAME" ] && [ -n "${DOCKERHUB_USERNAME_FROM_FILE:-}" ]; then
  DOCKERHUB_USERNAME="$DOCKERHUB_USERNAME_FROM_FILE"
fi

if [ -z "$DOCKERHUB_PASSWORD" ] && [ -n "${DOCKERHUB_PASSWORD_FROM_FILE:-}" ]; then
  DOCKERHUB_PASSWORD="$DOCKERHUB_PASSWORD_FROM_FILE"
fi

if [ -z "$DOCKERHUB_TOKEN" ] && [ -n "${DOCKERHUB_TOKEN_FROM_FILE:-}" ]; then
  DOCKERHUB_TOKEN="$DOCKERHUB_TOKEN_FROM_FILE"
fi

if [ ! -f "$VERSION_FILE" ]; then
  echo "0.0.1" > "$VERSION_FILE"
fi

CURRENT_VERSION="$(cat "$VERSION_FILE")"
if [[ ! "$CURRENT_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Error: $VERSION_FILE does not contain a valid semantic version (x.y.z)" >&2
  exit 1
fi

IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"
PATCH=$((PATCH + 1))
NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "$NEW_VERSION" > "$VERSION_FILE"

echo "Building ${IMAGE_NAME}:latest and ${IMAGE_NAME}:${NEW_VERSION}..."
docker build -t "${IMAGE_NAME}:latest" -t "${IMAGE_NAME}:${NEW_VERSION}" -f Dockerfile .

echo "Built ${IMAGE_NAME}:latest and ${IMAGE_NAME}:${NEW_VERSION}"

if [ "$DRY_RUN" = "1" ]; then
  echo "DRY_RUN=1 so skipping Docker Hub login and push."
  exit 0
fi

if [ -z "$DOCKERHUB_USERNAME" ]; then
  echo "Error: DOCKERHUB_USERNAME is required for push." >&2
  exit 1
fi

if [ -z "$DOCKERHUB_PASSWORD" ] && [ -z "$DOCKERHUB_TOKEN" ]; then
  echo "Error: DOCKERHUB_PASSWORD or DOCKERHUB_TOKEN is required for push." >&2
  exit 1
fi

if [ -n "$DOCKERHUB_PASSWORD" ]; then
  echo "$DOCKERHUB_PASSWORD" | docker login --username "$DOCKERHUB_USERNAME" --password-stdin
else
  echo "$DOCKERHUB_TOKEN" | docker login --username "$DOCKERHUB_USERNAME" --password-stdin
fi

docker push "${IMAGE_NAME}:latest"
docker push "${IMAGE_NAME}:${NEW_VERSION}"

echo "Pushed ${IMAGE_NAME}:latest and ${IMAGE_NAME}:${NEW_VERSION}"
