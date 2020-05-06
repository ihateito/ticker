FROM golang:alpine AS build-env
ADD ./src /src
RUN cd /src && go build -o app

FROM alpine
WORKDIR /go
COPY --from=build-env /src/app /go/
ENTRYPOINT ./app