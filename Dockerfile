FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/bot /app/bot

ENV SQLITE_PATH=/data/finbot.db
VOLUME ["/data"]

USER nonroot:nonroot
ENTRYPOINT ["/app/bot"]
