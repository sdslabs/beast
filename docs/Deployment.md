# Deployment

Beast can deploy a validated local challenge directory or synchronize the `challenges/` tree from configured SSH Git remotes. Active remotes are treated as sources of trusted build code.

The deployment pipeline validates configuration and paths, copies regular files into private staging, builds a generated/custom image or constrained Compose project under resource limits, selects a worker, starts the runtime, records identifiers/ports transactionally, and publishes normalized static assets when configured.

Remote synchronization and startup fail when an active remote/worker is unavailable; Beast does not silently operate with incomplete capacity. Periodic synchronization is opt-in with `beast run --periodic-sync`, and the scheduler prevents overlapping runs of the same task.

Before production deployment:

- review every setup script, Dockerfile, entrypoint, and Compose image as executable trusted code;
- pin base images by digest where reproducibility is required;
- keep flags/secrets out of Git and image layers, using validated file-backed environment values or runtime mechanisms;
- enforce worker firewall rules around allocated port ranges;
- terminate public challenge HTTP/TCP services appropriately and keep the Beast management API private over HTTPS;
- verify backup and restore procedures independently of controller shutdown.

Normal controller shutdown preserves deployed workloads. Use explicit challenge undeploy/purge operations when runtime removal is intended.
