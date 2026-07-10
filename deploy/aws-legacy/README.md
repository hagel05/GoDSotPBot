# AWS legacy (retired)

This directory holds the original AWS deployment, kept for reference only. It is
**not** part of the active build:

- `terraform/` — the Lambda + API Gateway + IAM + Secrets Manager stack.
- `_lambda/` — the original Lambda Go source. The leading underscore makes the
  Go toolchain ignore this directory, so it never compiles as part of the module
  (it imports AWS SDK packages that are no longer dependencies).

The bot now runs as a Docker container (see the repo root `Dockerfile` and
`deploy/docker-compose.example.yml`). Once the container is verified live, the
AWS resources can be torn down:

```sh
cd deploy/aws-legacy/terraform
terraform destroy
```

After teardown, this whole directory can be deleted — git history preserves it.
