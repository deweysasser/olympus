ARG GO_VERSION=1.19
FROM golang:${GO_VERSION} as builder
ARG PROGRAM=nothing
ARG VERSION=development

RUN mkdir /src /output

WORKDIR /src

COPY . .
RUN GOBIN=/output make install VERSION=$VERSION
RUN mkdir -p /received && chown 1000 /received



FROM gcr.io/distroless/base:latest
ARG PROGRAM=nothing

COPY --from=builder /output/${PROGRAM} /
COPY --from=builder /received /received
USER 1000
ENTRYPOINT [""]
CMD ["server"]
