
FROM golang:1.13.5 as builder

# arg deployment token
ARG BRANCH=master

ENV GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=https://goproxy.cn,direct
ENV LOGICPATH=message-center/cmd/logic
ENV MESSAGEPATH=message-center/cmd/message
ENV WORKSPACE=/data/workspace

WORKDIR $WORKSPACE

RUN cd message-center && echo Commit: `git log --pretty='%s%b%B' -n 1`
RUN cd $WORKSPACE/$LOGICPATH && go build -gcflags '-N -l' -o /tmp/logic-server .
RUN cd $WORKSPACE/$MESSAGEPATH && go build -gcflags '-N -l' -o /tmp/message-server .

FROM alpine:latest
COPY --from=builder /tmp/logic /
COPY --from=builder /tmp/message /
COPY --from=builder /data/workspacemessage-center/run.sh /

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