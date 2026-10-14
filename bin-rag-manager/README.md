# rag-manager

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

On `main`, `bin-rag-manager-test-auto`, `bin-rag-manager-build` and `bin-rag-manager-deploy` run right after the merge without an approval click, and the deploy goes through Komodo. On other branches `build-approval` still gates the test and build jobs and no deploy job is created. See `.circleci/auto-deploy-services.txt`.
