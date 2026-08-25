FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -o /out/crowdin-badge ./cmd/crowdin-badge

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/crowdin-badge /usr/local/bin/crowdin-badge
USER non-root
ENTRYPOINT ["/usr/local/bin/crowdin-badge"]
