#!/bin/bash

set -e

VERSION_FILE="VERSION"
DOCKERFILE_PATH="Dockerfile"
IMAGE_NAME="meeshotech/numerix"
PLATFORMS="linux/amd64,linux/arm64"  # 🧩 You can add more platforms here (e.g. linux/arm/v7)

# Ensure required tools
command -v git &>/dev/null || { echo "git not found"; exit 1; }
command -v docker &>/dev/null || { echo "docker not found"; exit 1; }
docker buildx version &>/dev/null || { echo "docker buildx not found"; exit 1; }

# Ensure buildx builder exists
if ! docker buildx inspect multiarch-builder &>/dev/null; then
    echo "🔧 Creating buildx builder 'multiarch-builder'..."
    docker buildx create --name multiarch-builder --use
    docker buildx inspect --bootstrap
else
    docker buildx use multiarch-builder
fi

# Enable binfmt for cross-platform builds (once per host)
docker run --privileged --rm tonistiigi/binfmt --install all

# Validate VERSION
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

echo "Creating Git tag: $GITHUB_TAG"
git add "$VERSION_FILE"
git commit -m "Bump version to $NEW_VERSION"
git push origin "$BRANCH_NAME"

git tag "$GITHUB_TAG"
git push origin "$GITHUB_TAG"

echo "🛠 Building Docker image for platforms: $PLATFORMS ..."
BUILD_CMD="docker buildx build \
  --platform $PLATFORMS \
  -f $DOCKERFILE_PATH \
  -t $IMAGE_NAME:$NEW_VERSION"

if [ "$PUSH_LATEST" = true ]; then
  BUILD_CMD+=" -t $IMAGE_NAME:latest"
fi

BUILD_CMD+=" --push ."

# Run the final build command
eval $BUILD_CMD

echo "✅ Done: pushed $IMAGE_NAME:$NEW_VERSION"
[ "$PUSH_LATEST" = true ] && echo "✅ Also tagged and pushed :latest" || echo "ℹ️ Skipped tagging :latest (non-release branch)"

echo "📦 Pushed image contains platforms:"
docker buildx imagetools inspect "$IMAGE_NAME:$NEW_VERSION" | grep -E 'Platform: ' || true