# syntax=docker/dockerfile:1.7
# Multi-stage Containerfile for go4pack
# Base distribution: Fedora 42

FROM registry.fedoraproject.org/fedora:42 AS base
LABEL org.opencontainers.image.source="https://example.com/go4pack" \
      org.opencontainers.image.title="go4pack" \
      org.opencontainers.image.description="Go file storage & analysis service with Next.js frontend" \
      org.opencontainers.image.licenses="MIT"

# Install build dependencies (Go toolchain, Node for building frontend)
# Using dnf module for Go; adjust version if Fedora repo Go mismatch with go.mod requirement
RUN rm -rf /etc/yum.repos.d/fedora-cisco-openh264.repo && dnf makecache

RUN dnf -y install golang nodejs pnpm && \
    dnf clean all && rm -rf /var/cache/dnf

# NEW: pnpm / npm registry mirror (override at build: --build-arg PNPM_REGISTRY=...)
ARG PNPM_REGISTRY=https://registry.npmmirror.com
ENV PNPM_REGISTRY=$PNPM_REGISTRY \
    NPM_CONFIG_REGISTRY=$PNPM_REGISTRY
# Global npmrc for all users
RUN echo "registry=${PNPM_REGISTRY}" > /etc/npmrc && \
    install -d -m 0755 /root && cp /etc/npmrc /root/.npmrc

ENV GO4PACK_ENV_TYPE=dev

WORKDIR /opt