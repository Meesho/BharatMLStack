#!/bin/bash

set -e

VERSION_FILE="VERSION"
IMAGE_BASE="meeshotech/skye"
PLATFORMS="linux/amd64,linux/arm64"

# Binaries and their Dockerfiles (order: admin, consumers, serving)
BINARIES=(admin consumers serving)
DOCKERFILES=(cmd/admin/Dockerfile cmd/consumers/Dockerfile cmd/serving/Dockerfile)

command -v git &>/dev/null || { echo "git not found"; exit 1; }
command -v docker &>/dev/null || { echo "docker not found"; exit 1; }
docker buildx version &>/dev/null || { echo "docker buildx not found"; exit 1; }

if ! docker buildx inspect multiarch-builder &>/dev/null; then
  echo "Creating buildx builder 'multiarch-builder'..."
  docker buildx create --name multiarch-builder --use
  docker buildx inspect --bootstrap
else
  docker buildx use multiarch-builder
fi

docker run --privileged --rm tonistiigi/binfmt --install all 2>/dev/null || true

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

git tag "skye/$GITHUB_TAG"
git push origin "skye/$GITHUB_TAG"

for i in "${!BINARIES[@]}"; do
  binary="${BINARIES[$i]}"
  DOCKERFILE="${DOCKERFILES[$i]}"
  IMAGE_NAME="${IMAGE_BASE}-${binary}"
  echo "Building $IMAGE_NAME for $PLATFORMS ..."
  BUILD_CMD="docker buildx build \
    --platform $PLATFORMS \
    -f $DOCKERFILE \
    -t $IMAGE_NAME:$NEW_VERSION"
  [ "$PUSH_LATEST" = true ] && BUILD_CMD+=" -t $IMAGE_NAME:latest"
  BUILD_CMD+=" --push ."
  eval $BUILD_CMD
  echo "Pushed $IMAGE_NAME:$NEW_VERSION"
done

[ "$PUSH_LATEST" = true ] && echo "Also tagged and pushed :latest for all images" || echo "Skipped :latest (non-release branch)"
