# Architecture

Service categories, inter-service communication patterns, deployment topology, and system-level dependencies for the VoIPbin monorepo.

| File | Description |
|---|---|
| [architecture-deep-dive.md](architecture-deep-dive.md) | 34-service catalog by category (call/media, AI, queue/routing, customer/agent, campaign, messaging, infrastructure), RabbitMQ RPC pattern, queue naming, configuration management, package layout, key dependencies, and Kubernetes deployment requirements |
| [service-dependency-graph.md](service-dependency-graph.md) | Cross-service dependency map showing which services call which, used to plan changes that cross multiple services |
