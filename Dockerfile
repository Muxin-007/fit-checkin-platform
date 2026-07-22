FROM harbor.seike.cn/builder/apline

ARG CONFIG_DIR=/etc/platform

# 安装 tzdata 包
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update \
    && apk upgrade \
    && apk --no-cache add tzdata

# 设置时区为亚洲上海（东八区）
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

RUN mkdir -p ${CONFIG_DIR}

COPY ./platform.yaml ${CONFIG_DIR}/platform.yaml
COPY ./bin/platform /usr/bin/platform

CMD ["/usr/bin/platform", "-f", "/etc/platform/platform.yaml"]
