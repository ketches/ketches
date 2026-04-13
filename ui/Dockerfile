# syntax=docker/dockerfile:1

# ─── Stage 1: Build ───────────────────────────────────────────────────────────
FROM node:22-alpine AS builder

WORKDIR /app

RUN corepack enable

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .
RUN pnpm run build

# ─── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM nginx:1.27-alpine

# Install gettext for envsubst
RUN apk add --no-cache gettext && \
    chown -R nginx:nginx /usr/share/nginx/html /etc/nginx/conf.d /var/cache/nginx /var/run /var/log/nginx

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx configs and custom entrypoint
COPY nginx.conf /etc/nginx/nginx.conf
COPY nginx.conf.template /etc/nginx/conf.d/default.conf.template
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh && \
    # Remove the default nginx config so ours takes over
    rm -f /etc/nginx/conf.d/default.conf

# Backend URL injected at runtime (override in docker run / compose)
ENV BACKEND_URL=http://api:8080

USER nginx

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
