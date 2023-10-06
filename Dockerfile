FROM golang:1.20 AS builder

RUN mkdir /src
WORKDIR /src

ADD . .
ARG VERSION
RUN \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 go build -ldflags="-X main.Version=$VERSION" -v -o /server ./cmd/server/

FROM scratch

LABEL org.opencontainers.image.source=https://github.com/pentops/o5-deploy-aws

COPY --from=builder /server /server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

ENTRYPOINT ["/server"]
