ARG SEMAPHORE_IMAGE
ARG SEMAPHORE_VERSION

FROM ${SEMAPHORE_IMAGE}:${SEMAPHORE_VERSION}

ARG TARGETARCH
ARG POWERSHELL_VERSION="7.5.0"

USER root

# Install dependencies
RUN apk add --no-cache \
    curl \
    ca-certificates \
    less \
    ncurses \
    krb5-libs \
    libgcc \
    libintl \
    libssl3 \
    libstdc++ \
    tzdata \
    userspace-rcu \
    zlib

RUN wget -O /tmp/powershell.tar.gz https://github.com/PowerShell/PowerShell/releases/download/v${POWERSHELL_VERSION}/powershell-${POWERSHELL_VERSION}-linux-${TARGETARCH/amd/x}.tar.gz

RUN mkdir -p /opt/microsoft/powershell/${POWERSHELL_VERSION} \
    && tar zxf /tmp/powershell.tar.gz -C /opt/microsoft/powershell/${POWERSHELL_VERSION} \
    && rm /tmp/powershell.tar.gz \
    && ln -s /opt/microsoft/powershell/${POWERSHELL_VERSION}/pwsh /usr/bin/pwsh

USER 1001