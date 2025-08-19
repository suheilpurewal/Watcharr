#
# Backend
#
FROM golang:1.24-alpine AS server

WORKDIR /server

# 1) deps first for cache
COPY server/go.* ./
RUN apk update && apk add --no-cache musl-dev gcc build-base
RUN go mod download

# 2) now all sources (includes groupview, arr, game, etc.)
COPY server/ ./

# 3) build
# CGO_CFLAGS: https://github.com/mattn/go-sqlite3/issues/1164#issuecomment-1635253695
RUN GOOS=linux CGO_ENABLED=1 CGO_CFLAGS="-D_LARGEFILE64_SOURCE" go build -o /watcharr

#
# Frontend
#
FROM node:20-alpine AS ui

WORKDIR /app
COPY package*.json vite.config.ts svelte.config.js tsconfig.json ./
COPY ./src ./src
COPY ./static ./static
RUN npm install && npm run build

#
# Production
#
FROM node:20-alpine AS runner

# app binary + UI
COPY --from=server /watcharr /watcharr
COPY --from=ui /app/build /ui
COPY --from=ui /app/package.json /app/package-lock.json /ui

# install only prod ui deps (ignore dev prepare scripts)
RUN cd /ui && npm ci --omit=dev --ignore-scripts=true

EXPOSE 3080
CMD ["/watcharr"]
