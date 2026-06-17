# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Version metadata is injected the same way the release workflow does it.
ARG VERSION=docker
ARG REVISION=unknown
ARG BUILD=docker
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.revision=${REVISION} -X main.build=${BUILD}" \
    -o /out/aiwolf-nlp-server .

# ---- runtime stage ----
# distroless/static is the smallest possible base for a CGO-free static binary.
# TTS (ffmpeg/VOICEVOX) is intentionally excluded to keep the image slim; run
# VOICEVOX as a separate container and point AIWOLF_TTS_HOST at it.
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/aiwolf-nlp-server /aiwolf-nlp-server
# Bundle the default game configs so the image runs out of the box; mount a
# volume over /config to override them.
COPY --from=build /src/config/*.yml /config/

EXPOSE 8080
ENTRYPOINT ["/aiwolf-nlp-server"]
CMD ["-c", "/config/default_5.yml"]
