FROM golang:1.24-alpine AS build
WORKDIR /src
COPY controlplane/go.mod controlplane/go.sum controlplane/
COPY onyxdb-go-sdk/go.mod onyxdb-go-sdk/go.sum onyxdb-go-sdk/
RUN cd onyxdb-go-sdk && go mod download
COPY controlplane/ controlplane/
COPY onyxdb-go-sdk/ onyxdb-go-sdk/
RUN cd onyxdb-go-sdk && CGO_ENABLED=0 GOWORK=off go build -o /onyxdb-perf ./cmd/perf

FROM alpine:3.20
COPY --from=build /onyxdb-perf /usr/local/bin/onyxdb-perf
ENTRYPOINT ["onyxdb-perf"]
