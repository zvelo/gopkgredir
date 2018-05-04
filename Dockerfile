FROM golang:alpine as build
RUN apk add --no-cache git
RUN go get github.com/magefile/mage
ARG VERSION
ARG GIT_COMMIT
COPY . /go/src/zvelo.io/gopkgredir
WORKDIR /go/src/zvelo.io/gopkgredir
RUN mage -v build

FROM alpine:latest
MAINTAINER Joshua Rubin <jrubin@zvelo.com>
ENTRYPOINT ["gopkgredir"]
COPY --from=build /go/src/zvelo.io/gopkgredir/gopkgredir /usr/local/bin/gopkgredir
