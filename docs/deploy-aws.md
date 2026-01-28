# Deploying the Hexiosec ASM Cloud Connector on AWS

Last updated: November 2025

This guide explains how to deploy the **Hexiosec ASM Cloud Connector** in your AWS environment using either:

- **AWS Lambda** — best for short-running scans (under 15 minutes)
- **AWS Fargate (ECS)** — best for longer scans or container-based workflows

Both deployment options use the same core configuration and provide the same resource discovery and synchronisation with Hexiosec ASM.

> **Note:**  
> This guide uses AWS CLI examples throughout. You can perform the same steps via the AWS Management Console if you prefer, using the same role names, policies, ARNs, and configuration values shown here.

---

## 1. Overview

The Cloud Connector identifies internet‑exposed resources within your AWS estate — such as EC2 instances, Route 53 domains, load balancers and more — and automatically adds them as **seeds** into your Hexiosec ASM scans.

You can deploy it in two ways:

- **Lambda:** Easiest to set up; best for smaller environments and shorter runs.
- **Fargate:** Allows longer execution times, supports container workflows, and avoids Lambda’s 15‑minute timeout.

The Cloud Connector loads configuration from a YAML file or environment variables and retrieves your API key securely from AWS Secrets Manager.

---

## 2. Prerequisites

Before deploying, ensure you have:

- An active **Hexiosec ASM** account and a valid API key
- AWS permissions for IAM, Lambda, ECS, EventBridge, Secrets Manager
- A Go toolchain installed if you plan to build the Lambda bundle from source (not required when using prebuilt releases)
- A VPC with subnets for Fargate deployments (if using ECS)
- AWS CLI installed (optional but recommended, as this guide uses CLI examples)

---

## 3. Configuration

The Cloud Connector reads its configuration from a `config.yml` file or from environment variables and/or AWS Systems Manager Parameter Store (SSM).

### Example `config.yml`

```yaml
scan_id: d580a913-318e-40e5-8442-7680909da530
seed_tag: cloud_connector
delete_stale_seeds: true

aws:
  enabled: true
  api_key_secret: asm-cloud-connector/api-key
  default_region: eu-west-2
  list_all_accounts: false
  accounts: []
  assume_role: CloudConnectorReadOnly
  services:
    check_ec2: true
    check_eip: false
    check_elb: false
    check_s3: false
    check_acm: false
    check_route53: true
    check_cloudfront: false
    check_api_gateway: false
    check_api_gateway_v2: false
    check_eks: false
    check_rds: false
    check_opensearch: false
    check_lambda: false

azure:
  enabled: false

gcp:
  enabled: false

http:
  retry_count: 3
  retry_base_delay: 1s
  retry_max_delay: 10s
```

### Environment variable mappings

| Environment Variable | Maps To              |
| -------------------- | -------------------- |
| `SCAN_ID`            | `scan_id`            |
| `SEED_TAG`           | `seed_tag`           |
| `DELETE_STALE_SEEDS` | `delete_stale_seeds` |

Environment variables override YAML values.

### Key Configuration Options

| Key                     | Description                                                                                                                                    |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `scan_id`               | ASM scan to receive discovered resources.                                                                                                      |
| `seed_tag`              | Label applied to all seeds created by this connector.                                                                                          |
| `delete_stale_seeds`    | Whether to remove resources no longer present in AWS.                                                                                          |
| `aws.api_key_secret`    | Path to the ASM API key secret in AWS Secrets Manager.                                                                                         |
| `aws.default_region`    | Default AWS region to query.                                                                                                                   |
| `aws.services.*`        | Toggles for individual AWS service checks.                                                                                                     |
| `aws.assume_role`       | Role name to assume when scanning multiple accounts. The Cloud Connector constructs the ARN as `arn:aws:iam::<ACCOUNT_ID>:role/<assume_role>`. |
| `aws.list_all_accounts` | When `true`, automatically enumerates all linked accounts under your organisation.                                                             |
| `aws.accounts`          | Explicit list of AWS account IDs to enumerate for resources.                                                                                   |
| `http.retry_*`          | Controls retry behaviour for API requests to Hexiosec ASM.                                                                                     |

> **Automatic account detection:**  
> If `list_all_accounts` is `true`, the Cloud Connector detects all organisation accounts automatically. Otherwise, specify account IDs under `accounts` and provide an `assume_role` name.

> **Validation rules:**
>
> - Either `AWS`, `Azure`, or `GCP` must be enabled.
> - `default_region` is required.
> - `assume_role` is required when `accounts` or `list_all_accounts` are defined.

---

## 4. Securely Storing the API Key

### Storing the API Key in AWS Secrets Manager (recommended for both Lambda and Fargate)

The Cloud Connector requires a Hexiosec ASM API key, which should be stored securely in **AWS Secrets Manager** for all deployment types.

```bash
aws secretsmanager create-secret \
  --name asm-cloud-connector/api-key \
  --description "Hexiosec ASM API key" \
  --secret-string "<YOUR_API_KEY>"
```

You will reference this secret in either your Lambda environment variables or your ECS Task Definition.

---

## 5. IAM Permissions

The Cloud Connector requires read-only access to a number of AWS services in order to discover external-facing resources.  
The policy below lists the **complete set of discovery permissions** needed for full AWS coverage (EC2, Route 53, S3, CloudFront, API Gateway, ACM, Lambda, EKS, RDS and OpenSearch).

This list is the same regardless of whether you deploy via **Lambda** or **Fargate**.  
Additional IAM requirements specific to each deployment method (such as trust policies and Secrets Manager/SSM access) are described in **section 6 (Lambda)** and **section 7 (Fargate)**.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "elasticloadbalancing:Describe*",
        "route53:List*",
        "s3:ListAllMyBuckets",
        "s3:GetBucketWebsite",
        "s3:GetBucketPublicAccessBlock",
        "s3:GetBucketAcl",
        "s3:GetBucketPolicy",
        "cloudfront:ListDistributions",
        "acm:ListCertificates",
        "apigateway:GET",
        "eks:ListClusters",
        "rds:Describe*",
        "es:ListDomainNames",
        "es:ListApplications",
        "lambda:ListFunctions",
        "lambda:GetFunctionUrlConfig"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## 6. Deployment Option A — AWS Lambda

Lambda is the simplest deployment model for smaller environments.

### 6.1 Build or download the Lambda bundle

You may build from source:

```bash
GOOS=linux GOARCH=amd64 \
go build -trimpath -tags lambda.norpc \
  -o bootstrap ./cmd/lambda

zip asm-cloud-connector.zip bootstrap config.yml
```

If you want the binary to report a version string taken from your Git tags, you can embed it at build time using `-ldflags`:

```bash
VERSION=$(git describe --tags --abbrev=0)

GOOS=linux GOARCH=amd64 \
go build -trimpath -tags lambda.norpc \
  -ldflags "-X github.com/hexiosec/asm-cloud-connector/internal/version.version=${VERSION}" \
  -o bootstrap ./cmd/lambda

zip asm-cloud-connector.zip bootstrap config.yml
```

Or download a prebuilt ZIP from the latest [GitHub Releases](https://github.com/hexiosec/asm-cloud-connector/releases).

### 6.2 Create the Lambda execution role

The Cloud Connector runs as a Lambda function and therefore needs an IAM role that Lambda can assume, with permissions to:

- discover AWS resources (Describe/List actions)
- retrieve the ASM API key from Secrets Manager
- write logs to CloudWatch
- optionally assume cross-account roles (if scanning multiple accounts)

This section explains how to create this role using the AWS CLI.

#### 6.2.1 Create the trust policy

Lambda must be allowed to assume the role.

Create the following trust policy:

```bash
cat > lambda-trust-policy.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
```

Create the role:

```bash
aws iam create-role \
  --role-name asm-cloud-connector-role \
  --assume-role-policy-document file://lambda-trust-policy.json
```

#### 6.2.2 Create the permissions policy

Now create the policy that grants read access to AWS services, access to Secrets Manager, CloudWatch logging, and optional cross-account role assumption.

```bash
cat > lambda-permissions.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "elasticloadbalancing:Describe*",
        "route53:List*",
        "s3:ListAllMyBuckets",
        "s3:GetBucketWebsite",
        "s3:GetBucketPublicAccessBlock",
        "s3:GetBucketAcl",
        "s3:GetBucketPolicy",
        "cloudfront:ListDistributions",
        "acm:ListCertificates",
        "apigateway:GET",
        "eks:ListClusters",
        "rds:Describe*",
        "es:ListDomainNames",
        "es:ListApplications",
        "lambda:ListFunctions",
        "lambda:GetFunctionUrlConfig"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "secretsmanager:GetSecretValue",
      "Resource": "arn:aws:secretsmanager:*:*:secret:asm-cloud-connector/api-key*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": "arn:aws:iam::*:role/CloudConnectorReadOnly"
    }
  ]
}
EOF
```

Attach the policy:

```bash
aws iam put-role-policy \
  --role-name asm-cloud-connector-role \
  --policy-name asm-cloud-connector-policy \
  --policy-document file://lambda-permissions.json
```

#### 6.2.3 Cross-Account Setup (Optional)

If the Cloud Connector needs to discover resources in multiple AWS accounts, each target account must contain a read-only IAM role that the Lambda execution role can assume.

The Cloud Connector automatically constructs the ARN using the account ID and your configured `assume_role` value.

For example:

```yaml
assume_role: CloudConnectorReadOnly
```

With account ID `123456789012`, the Cloud Connector will assume:

```
arn:aws:iam::123456789012:role/CloudConnectorReadOnly
```

### A. Create the Trust Policy (Target Account)

Replace `<CONNECTOR_ACCOUNT_ID>` with the AWS account ID where the Lambda function runs:

```bash
cat > cross-account-trust.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<CONNECTOR_ACCOUNT_ID>:role/asm-cloud-connector-role"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
```

### B. Create the Read-Only Discovery Policy (Target Account)

```bash
cat > cross-account-policy.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "elasticloadbalancing:Describe*",
        "route53:List*",
        "s3:ListAllMyBuckets",
        "cloudfront:ListDistributions",
        "acm:ListCertificates"
      ],
      "Resource": "*"
    }
  ]
}
EOF
```

### C. Create the Role and Attach the Policy (Target Account)

```bash
aws iam create-role \
  --role-name CloudConnectorReadOnly \
  --assume-role-policy-document file://cross-account-trust.json

aws iam put-role-policy \
  --role-name CloudConnectorReadOnly \
  --policy-name CloudConnectorReadOnlyPolicy \
  --policy-document file://cross-account-policy.json
```

### Notes

- Use the same role name in every target account (e.g. `CloudConnectorReadOnly`)
- Permissions should remain read-only

#### 6.2.4 Summary

By this point, you should have an `asm-cloud-connector-role` with:

- trust by `lambda.amazonaws.com`
- discovery permissions for AWS services
- Secrets Manager read access for the ASM API key
- CloudWatch logging permissions
- optional cross-account assumption permissions (`sts:AssumeRole`)

For broader automation patterns and security recommendations, see [section 8](#8-automation-and-security-recommendations).

### 6.3 Deploy the Lambda function

Using the AWS CLI:

```bash
aws lambda create-function \
  --function-name asm-cloud-connector \
  --runtime provided.al2 \
  --handler bootstrap \
  --zip-file fileb://asm-cloud-connector.zip \
  --timeout 300 \
  --role arn:aws:iam::<ACCOUNT_ID>:role/asm-cloud-connector-role
```

### 6.4 Schedule the Lambda

Use EventBridge to run the Cloud Connector periodically:

```bash
aws events put-rule \
  --name asm-cloud-connector-schedule \
  --schedule-expression "cron(0 23 * * ? *)"
```

Next, grant EventBridge permission to invoke the Lambda function and attach it as a target:

```bash
aws lambda add-permission \
  --function-name asm-cloud-connector \
  --statement-id asm-cloud-connector-schedule \
  --action "lambda:InvokeFunction" \
  --principal events.amazonaws.com \
  --source-arn arn:aws:events:<REGION>:<ACCOUNT_ID>:rule/asm-cloud-connector-schedule

aws events put-targets \
  --rule asm-cloud-connector-schedule \
  --targets "Id"="1","Arn"="arn:aws:lambda:<REGION>:<ACCOUNT_ID>:function:asm-cloud-connector"
```

---

## 7. Deployment Option B — AWS Fargate (ECS)

Fargate bypasses Lambda’s 15‑minute timeout and is suitable for organisations with large AWS estates.

This option uses the **public Docker Hub image**, so customers do **not** need to build or push images to ECR.

### 7.1 Store the Cloud Connector configuration in SSM

```bash
aws ssm put-parameter \
  --name "/asm/cloud-connector-config" \
  --value "$(cat config.yml)" \
  --type SecureString
```

### 7.2 Create the ECS task execution role

Fargate tasks require an IAM role that the ECS agent can assume.  
This role must allow the Cloud Connector to:

- discover AWS resources (Describe/List actions)
- read the Cloud Connector configuration from SSM
- retrieve the ASM API key from Secrets Manager
- write logs to CloudWatch
- optionally assume cross-account read-only roles (if scanning multiple accounts)

This section walks you through creating the ECS execution role using the AWS CLI.

#### 7.2.1 Create the trust policy

ECS tasks must be allowed to assume this role.

Create the trust policy:

```bash
cat > ecs-trust-policy.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
```

Create the role:

```bash
aws iam create-role \
  --role-name asm-cloud-connector-exec-role \
  --assume-role-policy-document file://ecs-trust-policy.json
```

#### 7.2.2 Create the permissions policy

Now attach the policy that grants:

- discovery permissions
- Secrets Manager access
- SSM access
- CloudWatch logging
- optional `sts:AssumeRole` for multi-account setups

```bash
cat > ecs-permissions.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "elasticloadbalancing:Describe*",
        "route53:List*",
        "s3:ListAllMyBuckets",
        "s3:GetBucketWebsite",
        "s3:GetBucketPublicAccessBlock",
        "s3:GetBucketAcl",
        "s3:GetBucketPolicy",
        "cloudfront:ListDistributions",
        "acm:ListCertificates",
        "apigateway:GET",
        "eks:ListClusters",
        "rds:Describe*",
        "es:ListDomainNames",
        "es:ListApplications",
        "lambda:ListFunctions",
        "lambda:GetFunctionUrlConfig"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "ssm:GetParameters",
      "Resource": "arn:aws:ssm:*:*:parameter/asm/cloud-connector-config"
    },
    {
      "Effect": "Allow",
      "Action": "secretsmanager:GetSecretValue",
      "Resource": "arn:aws:secretsmanager:*:*:secret:asm-cloud-connector/api-key*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": "arn:aws:iam::*:role/CloudConnectorReadOnly"
    }
  ]
}
EOF
```

Attach the policy:

```bash
aws iam put-role-policy \
  --role-name asm-cloud-connector-exec-role \
  --policy-name asm-cloud-connector-exec-policy \
  --policy-document file://ecs-permissions.json
```

#### 7.2.3 Cross-Account Setup (Optional)

If you want the Cloud Connector to discover resources across multiple AWS accounts, each target account must contain a read-only role the ECS task can assume — the same model as the Lambda deployment.

Use the same trust/permission structure from **6.2.3**, replacing the principal role ARN with:

```
arn:aws:iam::<CONNECTOR_ACCOUNT_ID>:role/asm-cloud-connector-exec-role
```

Use the same role name (`CloudConnectorReadOnly`) in every target account for consistency.

#### 7.2.4 Summary

By this stage, you should have an `asm-cloud-connector-exec-role` with:

- trust by `ecs-tasks.amazonaws.com`
- discovery permissions
- SSM + Secrets Manager read access
- CloudWatch logging permissions
- optional cross-account assumption permissions

For broader automation patterns and security recommendations, see [section 8](#8-automation-and-security-recommendations).

### 7.3 Create the ECS task definition

Example:

```bash
cat > task-def.json << 'EOF'
{
  "family": "asm-cloud-connector",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/asm-cloud-connector-exec-role",
  "taskRoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/asm-cloud-connector-exec-role",
  "containerDefinitions": [
    {
      "name": "connector",
      "image": "docker.io/hexiosec/asm-cloud-connector:latest",
      "essential": true,
      "secrets": [
        {
          "name": "CONNECTOR_CONFIG",
          "valueFrom": "arn:aws:ssm:<REGION>:<ACCOUNT_ID>:parameter/asm/cloud-connector-config"
        },
        {
          "name": "ASM_API_KEY",
          "valueFrom": "arn:aws:secretsmanager:<REGION>:<ACCOUNT_ID>:secret:asm-cloud-connector/api-key"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/asm-cloud-connector",
          "awslogs-region": "<REGION>",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
EOF
```

For simplicity, this example uses the `:latest` tag. In production, we recommend pinning the image to a specific version tag (for example, `hexiosec/asm-cloud-connector:v1.3.0`) so that task definitions are tied to a known connector version. You can still publish and maintain a `:latest` tag for testing or development, but use explicit version tags in your ECS task definitions whenever possible.

Update `<ACCOUNT_ID>`, `<REGION>`, and the image name to match your environment, and register it with ECS:

```bash
aws ecs register-task-definition \
  --cli-input-json file://task-def.json
```

After this command succeeds, you should see a `taskDefinitionArn` similar to:

```text
arn:aws:ecs:<REGION>:<ACCOUNT_ID>:task-definition/asm-cloud-connector:1
```

Before running the task, make sure the CloudWatch Logs group defined in the task definition exists. For the example above, create it with:

```bash
aws logs create-log-group \
  --log-group-name /ecs/asm-cloud-connector \
  --region <REGION>
```

Use the same region here as you configured in `awslogs-region` in the task definition.

### 7.4 Create ECS cluster

```bash
aws ecs create-cluster --cluster-name asm-cloud-connector
```

### 7.5 Create a scheduled Fargate task

EventBridge rule:

```bash
aws events put-rule \
  --name asm-cloud-connector-schedule \
  --schedule-expression "cron(0 23 * * ? *)"
```

You also need an IAM role that EventBridge can assume to run ECS tasks (for example, `asm-cloud-connector-events-role`) with permissions such as `ecs:RunTask` and `iam:PassRole` for the task role.

Then attach the ECS task as the target. First define the target JSON:

```bash
cat > ecs-target.json << 'EOF'
[
  {
    "Id": "1",
    "Arn": "arn:aws:ecs:<REGION>:<ACCOUNT_ID>:cluster/asm-cloud-connector",
    "RoleArn": "arn:aws:iam::<ACCOUNT_ID>:role/asm-cloud-connector-events-role",
    "EcsParameters": {
      "LaunchType": "FARGATE",
      "TaskDefinitionArn": "arn:aws:ecs:<REGION>:<ACCOUNT_ID>:task-definition/asm-cloud-connector",
      "NetworkConfiguration": {
        "awsvpcConfiguration": {
          "Subnets": ["subnet-1234567890abcdef0"],
          "AssignPublicIp": "ENABLED"
        }
      }
    }
  }
]
EOF
```

Then apply the target configuration:

```bash
aws events put-targets \
  --rule asm-cloud-connector-schedule \
  --targets file://ecs-target.json
```

Update `<REGION>`, `<ACCOUNT_ID>`, and subnet IDs to match your environment.

---

## 8. Automation and Security Recommendations

These recommendations apply to both deployment options (Lambda and Fargate) and help you keep the Cloud Connector secure and easier to operate at scale.

### 8.1 Automatic role provisioning with AWS Organizations (optional)

AWS does not automatically add IAM roles to new accounts, but you can use **AWS Organizations** and **EventBridge** to bootstrap the `CloudConnectorReadOnly` role (and any related policies) whenever a new account is created.

A common pattern is:

1. In your **management/organisation account**, create an **EventBridge rule** that listens for AWS Organizations events related to account creation (for example, `CreateAccountResult` or `InviteAccountToOrganization`).
2. Configure the rule to invoke a small **bootstrap Lambda function**.
3. In the bootstrap function:
   - Assume an appropriate role in the new member account.
   - Create or update the `CloudConnectorReadOnly` IAM role.
   - Attach the read-only discovery policy required by the Cloud
     Connector.
4. (Optional) Periodically run an audit job that:
   - Lists all organisation accounts.
   - Checks for the presence of the `CloudConnectorReadOnly` role.
   - Creates or repairs it if missing.

This approach ensures that:

- Every new account is ready for Cloud Connector discovery without manual setup.
- Both Lambda and Fargate deployments can immediately assume the same cross-account role name across all accounts.

### 8.2 Security best practices

To minimise risk and follow the principle of least privilege:

- **Use a dedicated AWS account for the Cloud Connector where possible.**  
  Running the Lambda function or Fargate tasks from a dedicated “security tooling” account limits the blast radius if credentials are compromised.

- **Restrict IAM permissions to what you actually use.**  
  If you are not scanning certain services (for example, EKS or OpenSearch), you can safely remove those permissions from the IAM policies.

- **Keep secrets isolated and tightly scoped.**  
   Limit access to the `asm-cloud-connector/api-key` secret and any related SSM parameters to the Cloud
  Connector’s execution role only.

- **Ensure outbound HTTPS access to Hexiosec ASM.**  
   Whether using Lambda or Fargate, the Cloud
  Connector must be able to reach the Hexiosec ASM API endpoint over HTTPS.  
   If you use a private VPC, ensure that NAT gateways, VPC endpoints, or proxies are correctly configured.

- **Maintain consistent role naming across accounts.**  
  Use the same `assume_role` name (for example, `CloudConnectorReadOnly`) everywhere. This keeps configuration simple and avoids per-account overrides.

- **Review logs and permissions regularly.**  
   Periodically review CloudWatch logs, IAM policies, and role usage to confirm the Cloud
  Connector is behaving as expected and only accessing what it needs.

## 9. Validating Your Deployment

After the first run:

- Check CloudWatch logs for sync messages
- Ensure ECS task exits with code 0 (Fargate)
- Ensure Lambda completes within expected time (Lambda)

### 9.1 Verify in Hexiosec ASM

In the Hexiosec ASM portal:

1. Navigate to the relevant **scan**.
2. Scroll down to the **Scan Seeds** widget on the Scan Overview page.
3. Confirm that new seeds tagged with `cloud_connector` (or your configured value) are present.
4. Check that subsequent scan runs include the AWS resources discovered by the Cloud Connector.

When the Cloud Connector adds or removes resources, the associated Hexiosec ASM scan will automatically re-run to analyse the updated set of seeds. You do not need to manually trigger a scan after each connector run; instead, use the ASM UI and logs to confirm that new or removed resources are reflected in the scan results.

---

## 10. Updating the Cloud Connector

### Lambda

Update the ZIP bundle:

```bash
aws lambda update-function-code \
  --function-name asm-cloud-connector \
  --zip-file fileb://asm-cloud-connector.zip
```

### Fargate

Edit the task definition to reference a newer container tag, ideally using a specific version rather than `latest`. For example:

```text
hexiosec/asm-cloud-connector:v1.3.0
```

We recommend:

- Using a versioned tag in your ECS task definitions (for example, `v1.3.0`).
- Optionally publishing a `latest` tag for testing and ad-hoc runs.
- Updating the task definition to point to the new versioned tag whenever you roll out an updated connector image.

---

## 11. Troubleshooting

| Issue                              | Cause                                      | Fix                                                                                                    |
| ---------------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| Authentication errors              | Invalid or missing API key/secret path     | Confirm the secret exists in Secrets Manager, matches `aws.api_key_secret`, and is readable.           |
| No resources found                 | Insufficient IAM permissions               | Review IAM role policies and ensure required `Describe`/`List` actions are present.                    |
| Timeout                            | Lambda execution too long                  | Increase Lambda timeout or switch to Fargate for larger environments.                                  |
| Cannot reach Hexiosec ASM endpoint | Restricted network egress from VPC/subnets | Allow outbound HTTPS to the Hexiosec ASM API (for example, via a NAT gateway, VPC endpoint, or proxy). |
| “AccessDenied” for SSM             | Missing `ssm:GetParameter` permission      | Add SSM read permissions for the Cloud Connector’s SSM parameter ARN.                                  |
| “AccessDenied” for secrets         | Missing `secretsmanager:GetSecretValue`    | Add Secrets Manager read permissions for the `asm-cloud-connector/api-key` secret ARN.                 |
| Missing seeds in ASM               | Wrong or misconfigured `scan_id`           | Confirm `scan_id` in config matches the target ASM scan and re-run the Cloud Connector.                |

---

For further help, contact **support@hexiosec.com**.
