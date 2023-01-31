FROM golang:1.19-alpine AS build
RUN apk add build-base git linux-headers
WORKDIR /work
COPY go.mod go.sum /work/

RUN go mod download
COPY ./ /work
RUN go build -o build/babylon-relayer main.go

FROM alpine:3.14 AS run
# the below utilities are added for testing purposes
RUN apk add bash curl jq

# copy wrapper script
COPY scripts/wrapper.sh /usr/bin/wrapper.sh

# Copy over binaries from the build-env
VOLUME /app
COPY --from=build /work/build/babylon-relayer /app/babylon-relayer
WORKDIR /app

# port for debugging server
EXPOSE 7597

ENTRYPOINT ["/usr/bin/wrapper.sh"]
