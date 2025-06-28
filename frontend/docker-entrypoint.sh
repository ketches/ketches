#!/bin/sh
set -e

if [ -n "$VITE_API_BASE_URL" ]; then
  echo "Replacing VITE_API_BASE_URL in env.js: $VITE_API_BASE_URL"
  sed -i "s#VITE_API_BASE_URL_PLACEHOLDER#$VITE_API_BASE_URL#g" /usr/share/nginx/html/env.js
fi

nginx -g 'daemon off;'
