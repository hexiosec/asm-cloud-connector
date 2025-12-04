# Deploying the Hexiosec ASM Cloud Connector on Google Cloud

_Last updated: December 2025_

This guide explains how to deploy the **Hexiosec ASM Cloud Connector** in your Google Cloud Platform (GCP) environment using **Cloud Run jobs**.

Cloud Run jobs are a good fit for the Cloud Connector because they:

- Run a container **to completion** (no HTTP server required).
- Support long‑running tasks (up to 24 hours).
- Integrate directly with **Secret Manager** for configuration and API keys.

The same core `config.yml` file used for AWS and Azure can be reused for GCP. The Cloud Connector then discovers internet‑exposed resources in your GCP projects and syncs them into Hexiosec ASM.

---

## 1. Overview

At a high level, deploying the Cloud Connector on GCP involves:

1. Storing your **Hexiosec ASM API key** in **Secret Manager**.
2. Storing the Cloud Connector **configuration file** in Secret Manager.
3. Creating a **Cloud Run job** that:
   - Uses the official Cloud Connector container image.
   - Reads the `API_KEY` and `CONNECTOR_CONFIG` from Secret Manager.
4. Granting the Cloud Run job service account permissions to:
   - Read the required secrets from **Secret Manager**.
   - Query the GCP resources the Cloud Connector needs to inspect (via Cloud Asset Inventory and the underlying APIs).
5. Scheduling the job using **Cloud Scheduler**.

---

## 2. Prerequisites

Before deploying, ensure you have:

- An active **Hexiosec ASM** account and a valid API key.
- GCP permissions for Cloud Run, Secret Manager, and creating service accounts.
- `gcloud` CLI installed and authenticated (optional but recommended, as this guide uses CLI examples).

---

## 3. Connector Configuration

The Cloud Connector reads its configuration from the environmental variable `CONNECTOR_CONFIG`.

### 3.1 Example

```yaml
scan_id: d580a913-318e-40e5-8442-7680909da530
seed_tag: cloud_connector
delete_stale_seeds: true

gcp:
  enabled: true
  projects:
    - projects/my-gcp-project-id
  services:
    check_dns_resource_record_set: true
    check_dns_managed_zone: true
    check_compute_instance: true
    check_compute_address: true
    check_storage_bucket: true
    check_cloud_function: true
    check_run_service: true
    check_run_domain_mapping: true
    check_api_gateway: true
    check_sql_instance: true
    check_compute_forwarding_rule: true
    check_compute_global_forwarding_rule: true
    check_compute_url_map: true
    check_app_engine_service: true
    check_gke_cluster: true
    check_certificates: true
```

### 3.2 Environment variable mappings

| Environment Variable | Maps To              |
| -------------------- | -------------------- |
| `SCAN_ID`            | `scan_id`            |
| `SEED_TAG`           | `seed_tag`           |
| `DELETE_STALE_SEEDS` | `delete_stale_seeds` |

Environment variables override YAML values.

### 3.4 Key Configuration Options

| Key                  | Description                                                |
| -------------------- | ---------------------------------------------------------- |
| `scan_id`            | ASM scan to receive discovered resources.                  |
| `seed_tag`           | Label applied to all seeds created by this connector.      |
| `delete_stale_seeds` | Whether to remove resources no longer present in GCP.      |
| `gcp.services.*`     | Toggles for individual GCP service checks.                 |
| `gcp.projects`       | List of GCP projects to enumerate for resources            |
| `http.retry_*`       | Controls retry behaviour for API requests to Hexiosec ASM. |

## 4. Create secrets in Google Cloud Secret Manager

We recommend storing both the **ASM API key** and the Cloud Connector **configuration** in **Secret Manager**, then exposing them to the job as environment variables.

Replace `PROJECT_ID` with your own project ID in the examples below.

### 4.1 ASM API key secret

Create a secret to hold your Hexiosec ASM API key:

```bash
gcloud secrets create asm-cloud-connector-api-key \
  --replication-policy="automatic" \
  --project=PROJECT_ID
```

```bash
printf '%s' "<YOUR_API_KEY>" | gcloud secrets versions add asm-cloud-connector-api-key \
  --data-file=- \
  --project=PROJECT_ID
```

### 4.2 Connector configuration secret

Assuming you have a local `config.yml` file:

```bash
gcloud secrets create asm-cloud-connector-config \
  --replication-policy="automatic" \
  --project=PROJECT_ID
```

```bash
gcloud secrets versions add asm-cloud-connector-config \
  --data-file=config.yml \
  --project=PROJECT_ID
```

We will map these secrets to the environment variables `API_KEY` and `CONNECTOR_CONFIG` in the Cloud Run job.

---

## 5. Service account and IAM permissions

Cloud Run jobs run as a **service account**. That service account must have permission to:

- Read the Cloud Connector secrets from **Secret Manager**.
- Call the GCP APIs used by the Cloud Connector (for example, via **Cloud Asset Inventory** and/or service‑specific read‑only roles).

### 5.1 Create a dedicated service account (recommended)

```bash
gcloud iam service-accounts create asm-cloud-connector-sa \
  --description="Service account for Hexiosec ASM GCP connector" \
  --display-name="ASM Cloud Connector" \
  --project=PROJECT_ID
```

### 5.2 Grant Secret Manager access

Grant the service account permission to read the secrets:

```bash
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:asm-cloud-connector-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### 5.3 Grant read access to GCP resources

The Cloud Connector requires **read-only** access to discover resources. For each project, you will need to add these roles:

- `roles/cloudasset.viewer` - Cloud Asset Inventory
  - to discover all services except Certificates
- `roles/certificatemanager.viewer` - Certificates
- Custom roles with permissions `storage.buckets.get` and `storage.buckets.getIamPolicy` - Storage buckets
  - A default role could be used but the only one with both these permissions is `roles/storage.admin` which has write permissions
  - We recommend adding a custom role with only the permissions required

Example binding role to service account:

```bash
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:asm-cloud-connector-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/cloudasset.viewer"
```

Example creating custom storage role:

```bash
cat > bucket-policy-reader.yaml << 'EOF'
title: "Bucket Policy Reader"
description: "Can view bucket metadata and IAM policy, but not object contents."
stage: "GA"
includedPermissions:
  - storage.buckets.get
  - storage.buckets.getIamPolicy
EOF
```

```bash
gcloud iam roles create bucketPolicyReader \
  --project=PROJECT_ID \
  --file=bucket-policy-reader.yaml
```

---

## 6. Create the Cloud Run job

The command below creates a Cloud Run job that:

- Runs the Cloud Connector container.
- Injects `API_KEY` and `CONNECTOR_CONFIG` from Secret Manager.
- Sets basic configuration via environment variables.

```bash
gcloud run jobs create asm-cloud-connector \
  --image=docker.io/hexiosec/asm-cloud-connector:latest \
  --region=REGION \
  --project=PROJECT_ID \
  --service-account=asm-cloud-connector-sa@PROJECT_ID.iam.gserviceaccount.com \
  --set-secrets=API_KEY=asm-cloud-connector-api-key:latest,CONNECTOR_CONFIG=asm-cloud-connector-config:latest \
  --set-env-vars=SCAN_ID=SCAN_ID \
  --max-retries=3 \
  --timeout=10m
```

- optional `SCAN_ID` should match the Hexiosec ASM scan you want to receive seeds; leave it unset if you've already specified it in the config file created earlier
- `REGION` is the region where the cloud run job will run (e.g. `europe-west2`)
- For simplicity, this example uses the `:latest` tag. In production, we recommend pinning the image to a specific version tag (for example, `hexiosec/asm-cloud-connector:v1.3.0`) so that task definitions are tied to a known connector version.
- `timeout` can be increased up to 24h, larger cloud estates will need longer execution times

> `--set-secrets` binds Secret Manager secrets directly to environment variables used by the Cloud Connector.

---

## 7. Schedule the Cloud Run job

```bash
gcloud scheduler jobs create http asm-cloud-connector-scheduler \
  --location=REGION \
  --schedule="0 23 * * *" \
  --uri="https://run.googleapis.com/v2/projects/PROJECT_ID/locations/REGION/jobs/asm-cloud-connector:run" \
  --http-method=POST \
  --oidc-service-account-email=asm-cloud-connector-sa@PROJECT_ID.iam.gserviceaccount.com
```

- `schedule` uses cron syntax and is when the job will run, the provided value will run the Cloud Connector every day at 11pm

## 8. Run and validate the job

### 8.1 Run the job once

```bash
gcloud run jobs execute asm-cloud-connector \
  --region=REGION \
  --project=PROJECT_ID \
  --wait
```

This runs the job immediately and waits for completion.

### 8.2 Check logs

In the Google Cloud Console:

1. Go to **Cloud Run → Jobs → asm-cloud-connector**.
2. Select the latest execution and open the **Logs** tab.

You should see log messages for:

- Loading configuration and API key.
- Enumerating GCP resources.
- Pushing seeds to Hexiosec ASM.

### 8.3 Verify in Hexiosec ASM

In the Hexiosec ASM portal:

1. Navigate to the relevant **scan**.
2. Scroll down to the **Scan Seeds** widget.
3. Confirm that new seeds tagged with `cloud_connector` are present.
4. Check that subsequent scan runs include the GCP resources discovered by the Cloud Connector.

---

## 9. Troubleshooting

| Issue                    | Possible Cause                                | Suggested Fix                                                                        |
| ------------------------ | --------------------------------------------- | ------------------------------------------------------------------------------------ |
| Authentication errors    | Invalid or missing API key / secret path      | Confirm the secret exists in Secrets Manager                                         |
| Missing resources in ASM | Insufficient IAM permissions on GCP resources | Check logs for warnings, the error detail will include the permission missing.       |
| Job times out            | Large GCP environment / slow API responses    | Increase `--task-timeout` and/or narrow the Cloud Connector’s scope in `config.yml`. |

---

For further help, contact **support@hexiosec.com**.
