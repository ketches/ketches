#!/bin/sh
set -e

if [ -n "$VITE_API_BASE_URL" ]; then
  sed -i "s#VITE_API_BASE_URL_PLACEHOLDER#$VITE_API_BASE_URL#g" /etc/nginx/nginx.conf
fi

nginx -g 'daemon off;'
