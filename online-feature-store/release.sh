#!/bin/bash

set -e

VERSION_FILE="VERSION"
PLATFORMS="linux/amd64,linux/arm64"

API_SERVER_IMG_NAME="meeshotech/online-feature-store-grpc-api-server"
API_SERVER_DOCKERFILE="cmd/api-server/DockerFile"

CONSUMER_IMG_NAME="meeshotech/online-feature-store-consumer"
CONSUMER_DOCKERFILE="cmd/consumer/DockerFile"

# Ensure required tools
command -v git &>/dev/null || { echo "git not found"; exit 1; }
command -v docker &>/dev/null || { echo "docker not found"; exit 1; }
docker buildx version &>/dev/null || { echo "docker buildx not found"; exit 1; }

# Setup builder
if ! docker buildx inspect multiarch-builder &>/dev/null; then
    echo "🔧 Creating buildx builder 'multiarch-builder'..."
    docker buildx create --name multiarch-builder --use
    docker buildx inspect --bootstrap
else
    docker buildx use multiarch-builder
fi

# Enable binfmt for cross-platform builds
docker run --privileged --rm tonistiigi/binfmt --install all

# Read and bump version
[ -f "$VERSION_FILE" ] || { echo "VERSION file not found"; exit 1; }

CURRENT_VERSION=$(cat "$VERSION_FILE")
echo "Current version: $CURRENT_VERSION"

SEMVER=${CURRENT_VERSION#v}
IFS='.' read -r MAJOR MINOR PATCH <<< "$SEMVER"

read -p "What type of version bump? (patch/minor/major): " BUMP_TYPE
case "$BUMP_TYPE" in
    patch) PATCH=$((PATCH + 1));;
    minor) MINOR=$((MINOR + 1)); PATCH=0;;
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0;;
    *) echo "Invalid bump type"; exit 1;;
esac

NEW_VERSION="v$MAJOR.$MINOR.$PATCH"
echo "Bumping version to: $NEW_VERSION"
echo "$NEW_VERSION" > "$VERSION_FILE"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)

if [[ "$BRANCH_NAME" =~ ^(main|master|develop|release-.*)$ ]]; then
    GITHUB_TAG="$NEW_VERSION"
    PUSH_LATEST=true
else
    GITHUB_TAG="$NEW_VERSION-$BRANCH_NAME"
    PUSH_LATEST=false
fi

# Git commit + tag
git add "$VERSION_FILE"
git commit -m "Bump version to $NEW_VERSION"
git push origin "$BRANCH_NAME"
git tag "$GITHUB_TAG"
git push origin "$GITHUB_TAG"
git tag sdks/go/$GITHUB_TAG
git push origin sdks/go/$GITHUB_TAG

# Function to build and verify each image
build_and_verify() {
  local name=$1
  local dockerfile=$2

  echo "🚀 Building $name for platforms: $PLATFORMS ..."
  BUILD_CMD="docker buildx build \
    --platform $PLATFORMS \
    -f $dockerfile \
    -t $name:$NEW_VERSION"

  if [ "$PUSH_LATEST" = true ]; then
    BUILD_CMD+=" -t $name:latest"
  fi

  BUILD_CMD+=" --push ."

  echo "Running: $BUILD_CMD"
  eval $BUILD_CMD

  echo "📦 Verifying platforms for $name:$NEW_VERSION ..."
  docker buildx imagetools inspect "$name:$NEW_VERSION" | grep -E 'Platform:' || {
    echo "❌ Image inspect failed for $name:$NEW_VERSION"
    exit 1
  }
  echo "✅ Successfully pushed $name:$NEW_VERSION"
  [ "$PUSH_LATEST" = true ] && echo "✅ Also tagged and pushed :latest" || echo "ℹ️ Skipped tagging :latest (non-release branch)"
}

# Build API and Consumer
build_and_verify "$API_SERVER_IMG_NAME" "$API_SERVER_DOCKERFILE"
build_and_verify "$CONSUMER_IMG_NAME" "$CONSUMER_DOCKERFILE"
