FROM golang:1.24-alpine AS build
WORKDIR /src
COPY controlplane/go.mod controlplane/go.sum controlplane/
COPY mnemo-go-sdk/go.mod mnemo-go-sdk/go.sum mnemo-go-sdk/
RUN cd mnemo-go-sdk && go mod download
COPY controlplane/ controlplane/
COPY mnemo-go-sdk/ mnemo-go-sdk/
RUN cd mnemo-go-sdk && CGO_ENABLED=0 GOWORK=off go build -o /mnemo-perf ./cmd/perf

FROM alpine:3.20
COPY --from=build /mnemo-perf /usr/local/bin/mnemo-perf
ENTRYPOINT ["mnemo-perf"]
