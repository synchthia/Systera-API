FROM alpine:latest

# Install Package
RUN set -x && \
    mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 && \

    # Set Timezone
    ln -sf /usr/share/zoneinfo/Asia/Tokyo /etc/localtime 

# Copy Bin
COPY main /usr/bin/systera-api
