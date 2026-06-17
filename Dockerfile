# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# バージョン情報はリリースワークフローと同じ方法で埋め込む。
ARG VERSION=docker
ARG REVISION=unknown
ARG BUILD=docker
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.revision=${REVISION} -X main.build=${BUILD}" \
    -o /out/aiwolf-nlp-server .

# ---- runtime stage ----
# CGO無効の静的バイナリ向けに最小の distroless/static を使う。
# イメージを軽量に保つためTTS（ffmpeg/VOICEVOX）は含めない。VOICEVOXは別コンテナで動かし
# 設定ファイルの tts_broadcaster.host で接続する。
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/aiwolf-nlp-server /aiwolf-nlp-server
# 既定の設定を同梱して単体で起動できるようにする。/config をマウントすれば上書き可能。
COPY --from=build /src/config/*.yml /config/

EXPOSE 8080
ENTRYPOINT ["/aiwolf-nlp-server"]
CMD ["-c", "/config/default_5.yml"]
