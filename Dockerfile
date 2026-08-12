FROM golang:1.26-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/tarkov-tk-bot ./cmd/tarkov-tk-bot

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/tarkov-tk-bot /tarkov-tk-bot

ENTRYPOINT ["/tarkov-tk-bot"]
CMD ["serve"]
