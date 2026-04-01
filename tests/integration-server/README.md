# Integration tests for the API server

This suite runs the integration tests for the API server

**Integration tests vs E2E tests**

An integration test is similar to an end-to-end test in that they
both test the software with dependencies included (e.g. with Postgres, Zot running)

The difference between an integration test and an end-to-end test is that:
- E2E tests use black-box testing to test workflows (higher level)
- Integration use white-box testing to test side effects (lower level)

E2E tests are higher level than integration tests:
- E2E tests may include more dependencies, the setup may be more complex
- Integration tests may include less dependencies, the setup may be less complex

Ideally, integration tests are more isolated than E2E tests

For example:
- Each test has their own OCI repository to avoid conflicts in Zot
- Each test is executed in a database transaction to avoid conflicts in Postgres

As a result, integration tests can be executed in parallel, which makes them
faster than E2E tests
