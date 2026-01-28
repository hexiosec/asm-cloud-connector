# Deploying the Hexiosec ASM Cloud Connector on Azure

_Last updated: January 2026_

This guide explains how to deploy the **Hexiosec ASM Cloud Connector** in your Azure environment using an **[Azure Container Apps Job](https://learn.microsoft.com/en-us/azure/container-apps/jobs?tabs=azure-cli)**. The connector runs as a container job on a user-configurable cron schedule and scales to zero when idle.

The Cloud Connector discovers internet-exposed resources in your Azure subscriptions and syncs them into Hexiosec ASM.

---

## 1. Overview

At a high level, deploying the Cloud Connector on Azure involves:

1. Creating a resource group and Key Vault.
2. Storing your Hexiosec ASM API key and configuration file in Key Vault.
3. Creating an Azure Container Apps Job that:
   - Runs the official Cloud Connector container image.
   - Reads `API_KEY` and `CONNECTOR_CONFIG` from Key Vault.
4. Granting the job Managed Identity permissions to:
   - Read the required secrets from Key Vault.
   - Query Azure resources via Azure Resource Graph.
5. Scheduling the job with a Container Apps cron schedule, which can be updated without rebuilding the image.

---

## 2. Prerequisites

Before deploying, ensure you have:

- An active **Hexiosec ASM** account and a valid API key.
- Azure permissions for Container Apps, Log Analytics, Key Vault, and role assignments.
- An Azure Container Apps environment (Linux) and a Log Analytics workspace.
- Container registry access if you use a private image (the example uses a public image).
- `az` CLI installed and authenticated (optional but recommended, as this guide uses CLI examples).

---

## 3. Connector Configuration

The Cloud Connector reads its configuration from the environment variable `CONNECTOR_CONFIG`.

### 3.1 Example

```yaml
scan_id: d580a913-318e-40e5-8442-7680909da530
seed_tag: cloud_connector
delete_stale_seeds: true

azure:
  enabled: true
  services:
    check_public_ip_addresses: true
    check_application_gateways: true
    check_application_gateway_certificates: true
    check_front_door_classic: true
    check_front_door_afd: true
    check_traffic_manager: true
    check_dns_zones: true
    check_dns_records: true
    check_storage_static_websites: true
    check_cdn_endpoints: true
    check_app_services: true
    check_sql_servers: true
    check_cosmos_db: true
    check_redis_cache: true
```

### 3.2 Environment variable mappings

| Environment Variable | Maps To              |
| -------------------- | -------------------- |
| `SCAN_ID`            | `scan_id`            |
| `SEED_TAG`           | `seed_tag`           |
| `DELETE_STALE_SEEDS` | `delete_stale_seeds` |

Environment variables override YAML values.

### 3.3 Key Configuration Options

| Key                  | Description                                                |
| -------------------- | ---------------------------------------------------------- |
| `scan_id`            | ASM scan to receive discovered resources.                  |
| `seed_tag`           | Label applied to all seeds created by this connector.      |
| `delete_stale_seeds` | Whether to remove resources no longer present in Azure.    |
| `azure.services.*`   | Toggles for individual Azure service checks.               |
| `http.retry_*`       | Controls retry behaviour for API requests to Hexiosec ASM. |

---

## 4. Store secrets in Azure Key Vault

We recommend storing both the **ASM API key** and the Cloud Connector **configuration** in **Key Vault**, then referencing them in the Container Apps Job.

Replace `SUBSCRIPTION_ID` with your Azure subscription ID.

### 4.1 Create a Key Vault

```bash
az group create --name asm-cloud-connector-rg --location uksouth

az keyvault create \
  --name asm-cloud-connector-kv \
  --resource-group asm-cloud-connector-rg \
  --location uksouth
```

If your Key Vault uses RBAC, grant yourself access to manage secrets:

```bash
az role assignment create \
  --assignee-object-id "$(az ad signed-in-user show --query id --output tsv)" \
  --assignee-principal-type User \
  --role "Key Vault Secrets Officer" \
  --scope /subscriptions/$SUBSCRIPTION_ID/resourceGroups/asm-cloud-connector-rg/providers/Microsoft.KeyVault/vaults/asm-cloud-connector-kv
```

### 4.2 Store the API key

```bash
az keyvault secret set \
  --vault-name asm-cloud-connector-kv \
  --name asm-cloud-connector-api-key \
  --value "<YOUR_API_KEY>"
```

### 4.3 Store the connector config

```bash
az keyvault secret set \
  --vault-name asm-cloud-connector-kv \
  --name asm-cloud-connector-config \
  --file config.yml
```

---

## 5. Create an Azure Container Apps Job

Create a Log Analytics workspace, a Container Apps environment, and a job that runs the Cloud Connector on a cron schedule.
Set your region explicitly; for London use `uksouth`.

### 5.1 Create a Log Analytics workspace

```bash
az monitor log-analytics workspace create \
  --name asm-cloud-connector-law \
  --resource-group asm-cloud-connector-rg \
  --location uksouth
```

Capture the workspace ID and shared key:

```bash
WORKSPACE_ID=$(az monitor log-analytics workspace show \
  --name asm-cloud-connector-law \
  --resource-group asm-cloud-connector-rg \
  --query customerId \
  --output tsv)

WORKSPACE_KEY=$(az monitor log-analytics workspace get-shared-keys \
  --name asm-cloud-connector-law \
  --resource-group asm-cloud-connector-rg \
  --query primarySharedKey \
  --output tsv)
```

### 5.2 Create a Container Apps environment

```bash
az containerapp env create \
  --name asm-cloud-connector-env \
  --resource-group asm-cloud-connector-rg \
  --location uksouth \
  --logs-workspace-id "$WORKSPACE_ID" \
  --logs-workspace-key "$WORKSPACE_KEY"
```

### 5.3 Create the Container Apps Job

```bash
az containerapp job create \
  --name asm-cloud-connector-job \
  --resource-group asm-cloud-connector-rg \
  --environment asm-cloud-connector-env \
  --trigger-type Schedule \
  --cron-expression "0 23 * * *" \
  --parallelism 1 \
  --replica-completion-count 1 \
  --replica-timeout 900 \
  --replica-retry-limit 3 \
  --image docker.io/hexiosec/asm-cloud-connector:latest \
  --mi-system-assigned \
  --secrets \
    api-key=keyvaultref:https://asm-cloud-connector-kv.vault.azure.net/secrets/asm-cloud-connector-api-key,identityref:system \
    connector-config=keyvaultref:https://asm-cloud-connector-kv.vault.azure.net/secrets/asm-cloud-connector-config,identityref:system \
  --env-vars \
    API_KEY=secretref:api-key \
    CONNECTOR_CONFIG=secretref:connector-config \
    SCAN_ID=<YOUR_SCAN_ID>
```

- Optional `SCAN_ID` should match the Hexiosec ASM scan you want to receive seeds; leave it unset if you already specified it in `config.yml`.
- For simplicity, this example uses the `:latest` tag. In production, pin the image to a specific version tag (for example, `hexiosec/asm-cloud-connector:v1.3.0`).
- Replica timeout is the maximum number of seconds the job is allowed to execute for, the example is set to 15 minutes (900 seconds).

### 5.4 Capture the managed identity principal ID

```bash
az containerapp job show \
  --name asm-cloud-connector-job \
  --resource-group asm-cloud-connector-rg \
  --query identity.principalId \
  --output tsv
```

> **Note:** Capture the principal ID from the output and use it for role assignments for `asm-cloud-connector-job`.

### 5.5 Assign RBAC roles

Using the principal ID from `asm-cloud-connector-job`, you can:

- Assign access per subscription (limit scope), or
- Assign access at a management group to cover all subscriptions in that group.

#### Option A: Limit to specific subscriptions

Repeat these role assignments for each subscription you want to scan:

```bash
az role assignment create \
  --assignee-object-id "$PRINCIPAL_ID" \
  --assignee-principal-type ServicePrincipal \
  --role "Reader" \
  --scope /subscriptions/$SUBSCRIPTION_ID
```

#### Option B: All subscriptions in a management group

If your organisation uses management groups, assign roles at the management group scope:

```bash
az role assignment create \
  --assignee-object-id "$PRINCIPAL_ID" \
  --assignee-principal-type ServicePrincipal \
  --role "Reader" \
  --scope /providers/Microsoft.Management/managementGroups/$MANAGEMENT_GROUP_ID
```

### 5.6 Grant Key Vault access

Use the same principal ID from `asm-cloud-connector-job`.

```bash
az role assignment create \
  --assignee-object-id "$PRINCIPAL_ID" \
  --assignee-principal-type ServicePrincipal \
  --role "Key Vault Secrets User" \
  --scope /subscriptions/$SUBSCRIPTION_ID/resourceGroups/asm-cloud-connector-rg/providers/Microsoft.KeyVault/vaults/asm-cloud-connector-kv
```

---

## 6. Configure the job schedule

Container Apps Jobs use a cron schedule that you can update without rebuilding or redeploying the image. We set a schedule in step 5.3 when we created the job, but it can be updated.

```bash
az containerapp job update \
  --name asm-cloud-connector-job \
  --resource-group asm-cloud-connector-rg \
  --cron-expression "0 23 * * *"
```

---

## 7. Run and validate

### 7.1 Run once

Run the job manually to validate the first execution:

```bash
az containerapp job start \
  --name asm-cloud-connector-job \
  --resource-group asm-cloud-connector-rg
```

### 7.2 Check logs

In the Azure portal:

1. Go to **Container Apps -> Jobs -> asm-cloud-connector-job**.
2. Select the latest execution and review the logs.

You should see log messages for:

- Loading configuration and API key.
- Enumerating Azure resources.
- Pushing seeds to Hexiosec ASM.

### 7.3 Verify in Hexiosec ASM

In the Hexiosec ASM portal:

1. Navigate to the relevant **scan**.
2. Scroll down to the **Scan Seeds** widget on the Scan Overview page.
3. Confirm that new seeds tagged with `cloud_connector` (or your configured value) are present.
4. Check that subsequent scan runs include the Azure resources discovered by the Cloud Connector.

When the Cloud Connector adds or removes resources, the associated Hexiosec ASM scan will automatically re-run to analyse the updated set of seeds. You do not need to manually trigger a scan after each connector run; instead, use the ASM UI and logs to confirm that new or removed resources are reflected in the scan results.

---

## 8. Troubleshooting

| Issue                       | Possible Cause                                   | Suggested Fix                                                                              |
| --------------------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| Job not running on schedule | Cron expression or job schedule disabled         | Verify the cron expression and confirm the job is enabled in Container Apps.               |
| Authentication errors       | Missing or invalid Key Vault access              | Verify Key Vault role assignment for the Container Apps Job identity.                      |
| Missing resources in ASM    | Insufficient RBAC permissions on Azure resources | Ensure Reader is assigned at subscription or management group scope.                       |
| Image pull failures         | Private registry access not configured           | Confirm registry credentials or use a public image.                                        |
| Job times out               | Large Azure estate / slow API responses          | Narrow scope in `config.yml` and, if needed, increase job limits or split by subscription. |

---

For further help, contact **support@hexiosec.com**.
