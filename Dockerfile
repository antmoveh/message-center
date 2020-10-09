
FROM golang:1.13.5 as builder

# arg deployment token
ARG BRANCH=master

ENV GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=https://goproxy.cn,direct
ENV WORKSPACE=/data/workspace/message-center

WORKDIR $WORKSPACE
ADD . .

RUN echo Commit: `git log --pretty='%s%b%B' -n 1`
RUN cd $WORKSPACE/cmd/logic && go build -gcflags '-N -l' -o /tmp/logic-server .
RUN cd $WORKSPACE/cmd/message && go build -gcflags '-N -l' -o /tmp/message-server .

FROM alpine
# RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/* && mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY --from=builder /tmp/logic-server /
COPY --from=builder /tmp/message-server /
COPY --from=builder /data/workspace/message-center/run.sh /

#Update time zone to Asia-Shanghai
COPY --from=builder /data/workspace/message-center/Shanghai /etc/localtime
RUN echo 'Asia/Shanghai' > /etc/timezone
RUN chmod +x /logic-server
RUN chmod +x /message-server
RUN chmod +x /run.sh

EXPOSE 7777
EXPOSE 7799

CMD ./run.sh

# docker build --build-arg BRANCH=develop -t image:tag .
