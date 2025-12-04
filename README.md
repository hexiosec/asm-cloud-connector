# Hexiosec ASM Cloud Connector

The Hexiosec ASM Cloud Connector enables seamless integration between your cloud environment and Hexiosec Attack Surface Management (ASM). It securely enumerates your cloud infrastructure, identifies external-facing resources — such as domains and IP addresses — and adds them as seeds to your ASM scan. This enhances asset visibility for ASM scans, ensuring comprehensive coverage of your cloud footprint is included in the scan.

This repository includes:

- Deployment guides for setting up the Cloud Connector within your chosen cloud provider.
- Testing CLI tools for advanced users who want to run local tests, perform manual synchronisation, or validate provider configuration.

The [ASM SDK GO](https://github.com/hexiosec/asm-sdk-go) communicates with the Hexiosec ASM API, which is fixed at [`https://asm.hexiosec.com/api`](https://asm.hexiosec.com/api).

## Important Notice

Using the Cloud Connector may **increase your scan size** and could **affect your account limits**.  
If you have any concerns about potential impact or costs, please refer to the [ASM pricing page](https://hexiosec.com/asm/pricing/) or contact **`support@hexiosec.com`** before deploying.

## Disclaimer

This software is provided **“as is”**, without warranty of any kind.
See [LICENSE](./LICENSE) file for details.

By deploying this Cloud Connector in your own cloud environment, you accept full responsibility for its configuration, operation, and any associated usage or infrastructure costs.

## Cloud Provider Documentation

Documentation for individual providers will be added here:

- **AWS** — [Deployment Guide](./docs/deploy-aws.md)
- **Azure** — _coming soon_
- **Google Cloud Platform (GCP)** — [Deployment Guide](./docs/deploy-gcp.md)

## Generating an API Key

The Cloud Connector requires an API key to authenticate with your Hexiosec ASM account.  
Follow the steps in our documentation to generate one:

[How to generate an API key](https://docs.hexiosec.com/asm/using-the-public-api)

## Network Connectivity

For the Cloud Connector to function correctly, ensure outbound access is allowed to:

| Destination                                                                                                                        | Purpose                                     |
| ---------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| [`https://app.hexiosec.com/api`](https://app.hexiosec.com/api)                                                                     | Communicates with the Hexiosec ASM platform |
| [`https://api.github.com/repos/hexiosec/asm-cloud-connector/tags`](https://api.github.com/repos/hexiosec/asm-cloud-connector/tags) | Checks for Cloud Connector version updates  |

If your environment enforces outbound firewall rules, whitelist these endpoints accordingly.

## Cloud Connector CLI (Main Tool)

The `cmd/connector` command is the main package for Hexiosec Cloud Connector.  
It authenticates with your cloud provider, collects internet-exposed resources, and synchronises it with the Hexiosec ASM platform.

For production deployments, see the cloud provider deployment guides:

- **AWS** — [Deployment Guide](./docs/deploy-aws.md)
- **Azure** — _coming soon_
- **Google Cloud Platform (GCP)** — [Deployment Guide](./docs/deploy-gcp.md)

### Configuration

The Cloud Connector reads configuration either from a `config.yml` file or from the `CONNECTOR_CONFIG` environment variable (full YAML content). It also supplements values with environment variables loaded from `.env` when present.

**Option A — `config.yml` on disk (default)**

Place `config.yml` next to the binary (or point `--config` to a custom path).

**Option B — `CONNECTOR_CONFIG` environment variable**

Provide the entire YAML configuration via an environment variable. This is useful for containerised deployments (for example, when AWS SSM injects the value into the task definition).

```bash
export CONNECTOR_CONFIG="$(cat <<'EOF'
scan_id: 00000000-0000-0000-0000-000000000000
seed_tag: cloud_connector
aws:
  enabled: true
  default_region: us-east-1
EOF
)"
./asm-cloud-connector
```

#### Base Configuration

| Field                 | YAML/env key                                                   | Purpose                                                                                                                | Notes/defaults                                                    |
| --------------------- | -------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| `ScanID`              | `scan_id`/`SCAN_ID`                                            | ASM scan that receives discovered resources (as seeds).                                                                | **Required**. Must be a valid scan UUID.                          |
| `SeedTag`             | `seed_tag`/`SEED_TAG`                                          | Label applied to all seeds created by the Cloud Connector.                                                             | Defaults to `cloud-connector` when not provided.                  |
| `DeleteStaleSeeds`    | `delete_stale_seeds`/`DELETE_STALE_SEEDS`                      | Controls whether seeds missing from the latest run, with the seed tag label, are deleted in ASM.                       | Defaults to `false` unless set in config or env.                  |
| `AWS`, `Azure`, `GCP` | `aws`, `azure`, `gcp` blocks (each with `enabled: true/false`) | Toggles discovery for each provider. At least one must be enabled.                                                     | Validation requires one provider block to be enabled.             |
| `Http.RetryCount`     | `http.retry_count`                                             | Number of retries for outbound HTTP requests to the endpoints listed in [Network Connectivity](#network-connectivity). | Defaults to `4` when omitted.                                     |
| `Http.RetryBaseDelay` | `http.retry_base_delay`                                        | Base delay between retries.                                                                                            | Defaults to `1s`. Accepts duration strings (`500ms`, `2s`, etc.). |
| `Http.RetryMaxDelay`  | `http.retry_max_delay`                                         | Upper bound on backoff delay.                                                                                          | Defaults to `5s`. Accepts duration strings (`500ms`, `2s`, etc.). |

Minimal example:

```yaml
scan_id: 00000000-0000-0000-0000-000000000000
seed_tag: cloud_connector
delete_stale_seeds: true
aws:
  enabled: true
```

#### AWS Configuration

| Field             | YAML/env key            | Purpose                                                                                                                                          | Notes/defaults                                                                                                                                          |
| ----------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Enabled`         | `aws.enabled`           | Toggles AWS discovery.                                                                                                                           | At least one cloud provider must be enabled overall.                                                                                                    |
| `DefaultRegion`   | `aws.default_region`    | AWS region used for authentication/initial API calls.                                                                                            | **Required.** Must be a valid AWS region code (e.g. `us-east-1`).                                                                                       |
| `APIKeySecret`    | `aws.api_key_secret`    | Name or Amazon Resource Name (ARN) of the AWS Secrets Manager secret that stores the ASM key. The secret should be stored in the default region. | Optional. Without this value, the env value is used                                                                                                     |
| `ListAllAccounts` | `aws.list_all_accounts` | When `true`, enumerates all AWS Organization accounts automatically.                                                                             | Requires the execution role to have `organizations:ListAccounts`. Mutually exclusive with manual `accounts` list.                                       |
| `Accounts[]`      | `aws.accounts`          | Explicit list of AWS account IDs to enumerate for resources.                                                                                     | Optional.                                                                                                                                               |
| `AssumeRole`      | `aws.assume_role`       | IAM role name assumed in each target account.                                                                                                    | **Required** when `list_all_accounts` is `true` or `accounts` are provided. The Cloud Connector assumes `arn:aws:iam::<account-id>:role/<assume_role>`. |
| `Services`        | `aws.services.*`        | Enables discovery for specific AWS services.                                                                                                     | Each flag defaults to `false`. See table below for individual toggles.                                                                                  |

AWS service toggles:

| Flag                | YAML key                            | Resources Collected (when enabled)                     |
| ------------------- | ----------------------------------- | ------------------------------------------------------ |
| `CheckEC2`          | `aws.services.check_ec2`            | EC2 instance public DNS names and IP addresses.        |
| `CheckEIP`          | `aws.services.check_eip`            | Elastic IP addresses.                                  |
| `CheckELB`          | `aws.services.check_elb`            | Load balancer DNS names and endpoints.                 |
| `CheckS3`           | `aws.services.check_s3`             | Public S3 bucket endpoints/websites.                   |
| `CheckACM`          | `aws.services.check_acm`            | ACM certificate domains and Subject Alternative Names. |
| `CheckRoute53`      | `aws.services.check_route53`        | Hosted zone domain names and records.                  |
| `CheckCloudFront`   | `aws.services.check_cloudfront`     | CloudFront distribution domains and origins.           |
| `CheckAPIGateway`   | `aws.services.check_api_gateway`    | API Gateway v1 custom/domain endpoints.                |
| `CheckAPIGatewayV2` | `aws.services.check_api_gateway_v2` | API Gateway v2 (HTTP/WebSocket) endpoints.             |
| `CheckEKS`          | `aws.services.check_eks`            | EKS cluster API endpoints.                             |
| `CheckRDS`          | `aws.services.check_rds`            | RDS instance and cluster endpoints.                    |
| `CheckOpenSearch`   | `aws.services.check_opensearch`     | OpenSearch domain endpoints.                           |
| `CheckLambda`       | `aws.services.check_lambda`         | Lambda Function URLs.                                  |

#### GCP Configuration

| Field        | YAML/env key     | Purpose                                          | Notes/defaults                                                                                              |
| ------------ | ---------------- | ------------------------------------------------ | ----------------------------------------------------------------------------------------------------------- |
| `Enabled`    | `gcp.enabled`    | Toggles GCP discovery.                           | At least one cloud provider must be enabled overall.                                                        |
| `Projects[]` | `gcp.projects`   | List of GCP projects to enumerate for resources. | **Required** when `gcp.enabled` is `true`. Must include at least one and be of the format `projects/123456` |
| `Services`   | `gcp.services.*` | Enables discovery for specific GCP services.     | Each flag defaults to `false`. See table below for individual toggles.                                      |

> To get the project number, you can use the command `gcloud projects list`

GCP service toggles:

| Flag                           | YAML key                                            | Resources Collected (when enabled)                               |
| ------------------------------ | --------------------------------------------------- | ---------------------------------------------------------------- |
| `CheckDNSResourceRecordSet`    | `gcp.services.check_dns_resource_record_set`        | Cloud DNS record sets: A/AAAA/CNAME subdomains and IPs.          |
| `CheckDNSManagedZone`          | `gcp.services.check_dns_managed_zone`               | Cloud DNS managed zone DNS names.                                |
| `CheckComputeInstance`         | `gcp.services.check_compute_instance`               | Compute Engine instance external (NAT) IP addresses.             |
| `CheckComputeAddress`          | `gcp.services.check_compute_address`                | Compute Engine external static IP addresses.                     |
| `CheckStorageBucket`           | `gcp.services.check_storage_bucket`                 | Public Cloud Storage buckets as URLs.                            |
| `CheckCloudFunction`           | `gcp.services.check_cloud_function`                 | HTTPS-triggered Cloud Functions URLs.                            |
| `CheckRunService`              | `gcp.services.check_run_service`                    | Cloud Run service URLs when IAM allows public access.            |
| `CheckRunDomainMapping`        | `gcp.services.check_run_domain_mapping`             | Cloud Run custom domain mappings.                                |
| `CheckAPIGateway`              | `gcp.services.check_api_gateway`                    | API Gateway default hostnames.                                   |
| `CheckSQLInstance`             | `gcp.services.check_sql_instance`                   | Cloud SQL public IP addresses.                                   |
| `CheckComputeForwardingRule`   | `gcp.services.check_compute_forwarding_rule`        | External forwarding rule IP addresses (regional load balancers). |
| `CheckComputeGlobalForwarding` | `gcp.services.check_compute_global_forwarding_rule` | External global forwarding rule IP addresses.                    |
| `CheckComputeURLMap`           | `gcp.services.check_compute_url_map`                | URL map host rules (domains/hostnames).                          |
| `CheckAppEngineService`        | `gcp.services.check_app_engine_service`             | App Engine default and service-specific `appspot.com` hostnames. |
| `CheckGKECluster`              | `gcp.services.check_gke_cluster`                    | Public GKE cluster API endpoints.                                |
| `CheckCertificates`            | `gcp.services.check_certificates`                   | Certificate Manager Subject Alternative Names and Domain names.  |

### Running Locally (Development/Test)

You can also build and run the Cloud Connector directly from source to validate configuration or test connectivity.

#### Prerequisites

- Go toolchain installed (tested with Go 1.22+)
- `API_KEY` environment variable set to a valid Hexiosec ASM API key
- Optional: `LOG_LEVEL` (defaults to `info`) and `--debug` flag for human-readable logs

#### Environment Setup

Environment variables can be loaded from the provided `.env.example` template:

```bash
cp .env.example .env
# Edit .env to include your real API key (and log level if desired)
```

The Cloud Connector CLI automatically loads `.env` (via `godotenv`) when it is present in the working directory.

#### Usage

```bash
go run ./cmd/connector --config ./config.yml [--debug]
```

- `--config` — Path to the YAML configuration file (defaults to `./config.yml`)
- `--debug` — Enables human-readable console logs

#### Examples

```bash
go run ./cmd/connector --config ./config.yml --debug
```

Logs show the Cloud Connector initialising, authenticating, collecting resources, and synchronising them with Hexiosec ASM.

## Testing CLI tools

This repository includes several command-line tools for testing and manual operation.
They are intended for developers and advanced users, and require checking out the source code locally and running commands directly using the Go toolchain.

These tools allow you to validate connectivity, configuration, and data synchronisation with the Hexiosec ASM platform.

### Manual Sync CLI

The `cmd/manual_sync` command pushes a one-off set of resources to the Hexiosec ASM service for a given scan.

#### Prerequisites

- Go toolchain installed (tested with Go 1.22+)
- `API_KEY` environment variable set to a valid Hexiosec ASM API key
- Optional: `LOG_LEVEL` (defaults to `info`) and `--debug` flag for human-readable logs

#### Environment Setup

Environment variables can be loaded from the provided `.env.example` template:

```bash
cp .env.example .env
# Edit .env to include your real API key (and log level if desired)
```

The manual sync CLI automatically loads `.env` (via `godotenv`) when it is present in the working directory.

#### Usage

```bash
go run ./cmd/manual_sync \
  --scan-id <SCAN_ID> \
  --seed-label <SEED_LABEL> \
  [--delete-stale-seeds=false] \
  <resource> [<resource> ...]
```

- `--scan-id` identifies the Hexiosec scan to update (required).
- `--seed-label` labels the resources within the scan (required).
- Provide each resource as a separate argument; at least one is required.
  - Supported resource types: FQDNs (e.g. `example.com`) and IPv4 addresses (e.g. `192.0.2.1`).
  - IPv6 addresses are not currently supported.
- Set `--delete-stale-seeds=false` to keep resources that are not in the provided list. It defaults to `true`.
- Add `--debug` for human-readable logs.

#### Examples

Synchronise two hostnames into scan without removing existing seeds:

```bash
go run ./cmd/manual_sync \
  --scan-id 00000000-00000000-00000000-00000000 \
  --seed-label "Manual Upload" \
  --delete-stale-seeds=false \
  example.com api.example.com
```

Run with structured debug logging and remove stale seeds by default:

```bash
LOG_LEVEL=debug go run ./cmd/manual_sync \
  --debug \
  --scan-id 00000000-00000000-00000000-00000000 \
  --seed-label "Nightly Sync" \
  corp.example.net 192.0.2.0
```

### Version Check CLI

The `cmd/check_version` command checks the current Hexiosec Cloud Connector version against the latest published tag in GitHub and logs whether a newer version is available.

#### Prerequisites

- Go toolchain installed (tested with Go 1.22+)
- Optional: `.env` file with `LOG_LEVEL` and other environment configuration
- Optional: `--debug` flag for human-readable console output

#### Environment Setup

Environment variables can be loaded from the provided `.env.example` template:

```bash
cp .env.example .env
# Edit .env to include your preferred log level
```

The version check CLI automatically loads `.env` (via `godotenv`) when it is present in the working directory.

#### Usage

```bash
go run -ldflags "-X github.com/hexiosec/asm-cloud-connector/internal/version.version=<VERSION>" \
  ./cmd/check_version [--debug]
```

- --debug enables human-readable console logs (useful for local testing).
- The version variable is injected at build time using -ldflags.

#### Examples

Run a version check with debug output enabled:

```bash
go run -ldflags "-X github.com/hexiosec/asm-cloud-connector/internal/version.version=v1.0.0" \
  ./cmd/check_version --debug
```

## Versioning and Support Policy

The Hexiosec ASM Cloud Connector follows a structured versioning and support approach to ensure stability and security.

- **Supported Versions**

  - The current release and the previous minor release are actively supported.
  - Both receive security and critical bug fixes where applicable.
  - This rule applies across major versions — for example:
    - When `1.1.0` and `2.0.0` are current, both are supported.
    - When `2.1.0` is released, support continues for `2.1.0` and `2.0.0`.

- **Patch Releases**

  - Patch versions are reserved for bug fixes and security updates.
  - They do **not** introduce breaking changes or alter expected behaviour.

- **Major Versions**
  - Major releases may include breaking changes.
  - The previous major version remains supported under the same “current + previous minor” rule to allow safe migration.

Version numbers follow [Semantic Versioning](https://semver.org/):  
`MAJOR.MINOR.PATCH` (e.g. `1.3.2`)

## Changelog

Details of all notable changes are documented in the [CHANGELOG.md](./CHANGELOG.md) file.

Each entry includes the release date, version number, and a summary of the change.

## Troubleshooting

If you encounter issues during setup or operation, please reach out to: **`support@hexiosec.com`**

Include:

- The cloud provider you’re deploying to
- Any relevant logs or error messages

### Verifying Your API Key

If you see authentication errors when running CLI commands, it may indicate an invalid or incorrect API key.

You can quickly verify your key by making a simple GET request using your preferred tool (e.g. `curl`, Insomnia, or Postman):

```bash
curl -X GET "https://asm.hexiosec.com/api/auth" \
 -H 'accept: application/json' \
 -H 'x-hexiosec-api-key: <API_KEY>'
```

A successful response (HTTP 200) confirms your key is valid and correctly formatted.
