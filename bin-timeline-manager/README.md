# timeline-manager

<!-- Updated dependencies: 2026-03-15 -->

# Deploy

On `main`, `bin-timeline-manager-test-auto`, `bin-timeline-manager-build` and `bin-timeline-manager-deploy` run right after the merge without an approval click, and the deploy goes through Komodo. On other branches `build-approval` still gates the test and build jobs and no deploy job is created. See `.circleci/auto-deploy-services.txt`.
