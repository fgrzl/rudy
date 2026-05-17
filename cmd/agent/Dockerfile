FROM --platform=$BUILDPLATFORM golang:1.25.6-trixie AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN : "${TARGETOS:=linux}" "${TARGETARCH:=amd64}" \
	&& CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -trimpath -ldflags="-s -w" -o /out/localagent ./cmd/agent

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/localagent /usr/local/bin/localagent

ENV LOCALAGENT_HTTP_ADDR=:8080 \
	LOCALAGENT_DATA_DIR=/data \
	LOCALAGENT_WORKSPACE_DIR=/workspace

EXPOSE 8080

VOLUME ["/data", "/workspace"]

ENTRYPOINT ["/usr/local/bin/localagent"]
