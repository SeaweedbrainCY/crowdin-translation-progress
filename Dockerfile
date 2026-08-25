FROM golang:1.27-alpine AS build
WORKDIR /src
COPY src/go.mod ./
COPY src/main.go ./
RUN CGO_ENABLED=0 go build -o /out/crowdin-badge .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/crowdin-badge /usr/local/bin/crowdin-badge
USER non-root
ENTRYPOINT ["/usr/local/bin/crowdin-badge"]
