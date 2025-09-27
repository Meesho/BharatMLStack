#!/bin/sh
set -e

# Generate env.js dynamically based on container env var
cat <<EOF > /usr/share/nginx/html/env.js
window.env = {
  REACT_APP_HORIZON_BASE_URL: "${REACT_APP_HORIZON_BASE_URL:-http://localhost:8082}",
};
EOF

echo "✅ Generated env.js with REACT_APP_HORIZON_BASE_URL=${REACT_APP_HORIZON_BASE_URL}"

# Start nginx
exec nginx -g "daemon off;"