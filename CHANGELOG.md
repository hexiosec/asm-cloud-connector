# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.1.0]

- Added `ignore_list` configuration (or comma-separated `IGNORE_LIST`) to exclude specific discovered seeds across all cloud providers. Matching uses normalised domains and IP addresses; existing connector-tagged seeds are removed only when `delete_stale_seeds` is enabled.
- Building from source now requires Go 1.26 or newer. Container and Lambda deployments require no migration.
- Dependency updates

## [2.0.0]

- Added AWS tag collection for discovered assets and propagation of those tags to ASM seed creation, with support across the current AWS resource discovery set.
- Added `aws.populate_tags` configuration, defaulting to `true`, to let deployments disable AWS tag collection and the related tag-read API calls.
- Changed internal resource handling from raw strings to `Asset{Name, Tags}` objects so provider metadata can flow through to connector seed creation.
- Updated connector deduplication and normalisation to preserve and merge asset tags before appending the configured seed tag.
- Updated AWS deployment documentation with `aws.populate_tags`, required tag-read IAM permissions.
- Updated AWS, Azure, GCP, and supporting SDK dependencies.

## [1.3.0]

- Added Azure connector deployment documentation

## [1.2.0]

- Added Azure connector

## [1.1.0]

- Minor fixes to AWS deployment guide
- Added GCP cloud connector
- Updated readme with GCP config options
- Added the name of the cloud service being used to the logs
- Fix to allow AWS cloud connector to continue when get resources fail to assume the role on an account

## [1.0.0]

- Initial release
- Added Hexiosec ASM API cloud connector
- Added AWS cloud connector
