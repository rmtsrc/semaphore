# Semaphore UI auto build

[![Auto Merge Latest Release](https://github.com/rmtsrc/semaphore/actions/workflows/auto-merge-release.yml/badge.svg)](https://github.com/rmtsrc/semaphore/actions/workflows/auto-merge-release.yml)
[![GitHub Container Registry](https://img.shields.io/badge/ghcr-semaphore-blue?logo=docker)](https://github.com/rmtsrc/semaphore/pkgs/container/semaphore)

This fork automatically builds a modified version of [Semaphore UI](https://github.com/semaphoreui/semaphore) from the latest release, based on the patches listed in [auto-merge-release.yml](.github/workflows/auto-merge-release.yml#L67-L70).

These patches include:

- [fix: use locale datetime on full dates](https://github.com/semaphoreui/semaphore/pull/2442)
- [feat(enhancement): use system dark mode](https://github.com/semaphoreui/semaphore/pull/1283)
