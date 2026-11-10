# 1. Use C4 and Structurizr for architecture

Date: 2026-11-07

## Status

Accepted

## Context

CloudChamber is a multi-service platform with a layered Go layout and deployable infrastructure. Architecture was spread across informal diagrams and long standalone markdown, which drifts from the code.

## Decision

Describe structure with the C4 model and keep that model in Structurizr alongside the repo.

Treat Structurizr as the single place for architecture views, supplementary documentation, and decision records, not parallel copies elsewhere.

ADRs state enduring why and principles, including layer dependency direction and data flow across layers. Operational how-to stays in the README, makefile, and agent skills.

## Consequences

Context, containers, deployment, and narrative docs stay aligned when the system changes.

Reviewers and new contributors look in one system instead of hunting files.

This ADR should not need updates when tooling or paths change.
