FROM golang:1.25.6-bookworm AS build

WORKDIR /src

RUN apt-get update \
  && apt-get install -y --no-install-recommends graphviz libgraphviz-dev \
  && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /out/guest-emulator ./internal

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates graphviz \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/guest-emulator /app/guest-emulator
COPY docs /app/docs

ENTRYPOINT ["/app/guest-emulator"]
