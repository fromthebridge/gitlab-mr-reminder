FROM golang:alpine AS build
RUN apk update && apk add ca-certificates
ADD . /
RUN cd /src && go build -o gitlab-mr-reminder

FROM alpine:latest
WORKDIR /bin
COPY --from=build /src/gitlab-mr-reminder /bin/
COPY --from=build /etc/ssl/certs /etc/ssl/certs
ENTRYPOINT ./gitlab-mr-reminder