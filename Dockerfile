FROM node:22-bookworm-slim AS admin-build

WORKDIR /src

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/admin/package.json apps/admin/package.json

RUN corepack enable && pnpm install --frozen-lockfile

COPY apps/admin apps/admin
RUN pnpm --dir apps/admin build


FROM golang:1.23-bookworm AS server-build

WORKDIR /src/server

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/fenghuolun .


FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai \
    FENGHUOLUN_HTTP_ADDR=0.0.0.0:8088 \
    FENGHUOLUN_SQLITE_PATH=/var/lib/fenghuolun/fenghuolun.db \
    FENGHUOLUN_ADMIN_DIR=/app/admin

WORKDIR /app
COPY --from=server-build /out/fenghuolun /app/fenghuolun
COPY --from=admin-build /src/apps/admin/dist /app/admin

EXPOSE 8088
ENTRYPOINT ["/app/fenghuolun"]
