# Deployment contracts

HackForger keeps only business-neutral deployment primitives in this public
repository. Real hosts, domains, accounts, filesystem layouts, runbooks,
runtime evidence, and branded overlays belong to the corresponding private
business repository.

For externally owned static content, use [`content/README.md`](content/README.md).
The private repository supplies a tracked configuration, a tracked content
tree, and a tracked checksum manifest; the public publisher validates and
hydrates that material without compiling or restarting HackForger.

A business-specific release wrapper should perform the following sequence:

1. verify both repositories are clean, pushed, and at the reviewed revisions;
2. run the public repository boundary check and application release gates;
3. back up application data using the private environment runbook;
4. publish each supported private-content mount with the generic publisher;
5. verify content hashes, public behavior, and independent deployment markers;
6. store the production report and screenshots in the private repository.

Never copy private overlays into the public Git index. Do not use
`rsync --delete` for externally owned content, and do not treat an application
binary deployment as a substitute for content hydration.

## Runtime overlay compatibility

Treat a runtime template override and the application data contract it consumes
as one versioned release unit. When a public handler or template key changes,
the private overlay must be updated and staged before the new process starts;
rollback must restore both revisions together. Production templates are compiled
when the renderer initializes, so replacing the on-disk override before the
reviewed restart does not change the already-running process. Never publish a
new binary against an older, unverified private template overlay.
