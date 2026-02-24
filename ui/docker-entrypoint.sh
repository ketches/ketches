#!/bin/sh
# Substitute only ${BACKEND_URL} so that nginx variables ($host, $uri, etc.)
# are left untouched, then hand off to the official nginx entrypoint.
set -e

envsubst '${BACKEND_URL}' \
  < /etc/nginx/conf.d/default.conf.template \
  > /etc/nginx/conf.d/default.conf

exec "$@"
