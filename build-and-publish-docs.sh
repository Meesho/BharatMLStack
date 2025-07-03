#!/bin/bash

set -e  # Exit immediately if a command fails

cd docs-src

echo "Building documentation..."
if ! npm run build; then
  echo "❌ Build failed. Aborting."
  exit 1
fi

echo "Clearing old docs..."
rm -rf ../docs/*

echo "Copying new docs..."
cp -r build/* ../docs/

echo "✅ Documentation built and copied successfully."
echo "TO DO: create PR to develop and merge to main to update the doc site"
