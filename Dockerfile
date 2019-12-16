FROM golang:1.12.9-alpine as base
RUN apk --update upgrade
RUN apk --no-cache add tzdata make bash curl g++ git
RUN rm -rf /var/cache/apk/*
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo "Asia/Shanghai" > /etc/timezone

FROM base as builder
WORKDIR $GOPATH/src/kubecloud
COPY . .
RUN CGO_ENABLED=1 INSTALL_DIR=/kubecloud make install clean

FROM base
WORKDIR /kubecloud
COPY --from=builder /kubecloud .
EXPOSE 8080
CMD ["./kubecloud"]
