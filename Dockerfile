FROM golang:1.27-alpine AS build
WORKDIR /src
COPY src/go.mod ./
COPY src/main.go ./
RUN CGO_ENABLED=0 go build -o /out/crowdin-badge .



FROM gcr.io/distroless/static-debian13
COPY --from=build /out/crowdin-badge /usr/local/bin/crowdin-badge
# nosemgrep: dockerfile.security.missing-user-entrypoint.missing-user-entrypoint Intended for github action which run with a specific and import docker user
ENTRYPOINT ["/usr/local/bin/crowdin-badge"]
