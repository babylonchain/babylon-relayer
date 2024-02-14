# Changelog

## [Unreleased](https://github.com/babylonchain/babylon-relayer/tree/HEAD)

[Full Changelog](https://github.com/babylonchain/babylon-relayer/compare/v0.4.0...HEAD)

**Merged pull requests:**

- migrating private commits [\#29](https://github.com/babylonchain/babylon-relayer/pull/29) ([SebastianElvis](https://github.com/SebastianElvis))
- phase1: removing the need of `paths` for phase1 chains [\#25](https://github.com/babylonchain/babylon-relayer/pull/25) ([SebastianElvis](https://github.com/SebastianElvis))
- CI: Build and push images to ECR [\#24](https://github.com/babylonchain/babylon-relayer/pull/24) ([filippos47](https://github.com/filippos47))
- fix: fixing nil pointer bug upon bumping relayer dependency [\#22](https://github.com/babylonchain/babylon-relayer/pull/22) ([SebastianElvis](https://github.com/SebastianElvis))
- chore: bump relayer version to v0.2.3 [\#21](https://github.com/babylonchain/babylon-relayer/pull/21) ([vitsalis](https://github.com/vitsalis))
- fix: fix retries and metrics [\#20](https://github.com/babylonchain/babylon-relayer/pull/20) ([SebastianElvis](https://github.com/SebastianElvis))
- ci: add CI [\#19](https://github.com/babylonchain/babylon-relayer/pull/19) ([SebastianElvis](https://github.com/SebastianElvis))
- chore: move `createClientIfNotExist` [\#18](https://github.com/babylonchain/babylon-relayer/pull/18) ([SebastianElvis](https://github.com/SebastianElvis))
- chore: cleanup example config files [\#17](https://github.com/babylonchain/babylon-relayer/pull/17) ([SebastianElvis](https://github.com/SebastianElvis))
- feat: store client ID in LevelDB rather than config file [\#16](https://github.com/babylonchain/babylon-relayer/pull/16) ([SebastianElvis](https://github.com/SebastianElvis))
- feat: replace in-process lock with a filesystem lock for running concurrent relayers [\#15](https://github.com/babylonchain/babylon-relayer/pull/15) ([SebastianElvis](https://github.com/SebastianElvis))
- chore: bump relayer dependency [\#14](https://github.com/babylonchain/babylon-relayer/pull/14) ([SebastianElvis](https://github.com/SebastianElvis))
- fix: write client ID back to config file on-the-fly [\#13](https://github.com/babylonchain/babylon-relayer/pull/13) ([SebastianElvis](https://github.com/SebastianElvis))

## [v0.4.0](https://github.com/babylonchain/babylon-relayer/tree/v0.4.0) (2024-02-08)

[Full Changelog](https://github.com/babylonchain/babylon-relayer/compare/v0.3.0...v0.4.0)

**Closed issues:**

- Update to latest `main` branch of relayer [\#28](https://github.com/babylonchain/babylon-relayer/issues/28)

## [v0.3.0](https://github.com/babylonchain/babylon-relayer/tree/v0.3.0) (2023-06-07)

[Full Changelog](https://github.com/babylonchain/babylon-relayer/compare/v0.2.0...v0.3.0)

## [v0.2.0](https://github.com/babylonchain/babylon-relayer/tree/v0.2.0) (2023-02-03)

[Full Changelog](https://github.com/babylonchain/babylon-relayer/compare/v0.1.0...v0.2.0)

**Merged pull requests:**

- fix: override retry config, enforce large retry times, and fix min gas amount [\#12](https://github.com/babylonchain/babylon-relayer/pull/12) ([SebastianElvis](https://github.com/SebastianElvis))
- infra: adding wrapper with env variable for Dockerfile [\#11](https://github.com/babylonchain/babylon-relayer/pull/11) ([SebastianElvis](https://github.com/SebastianElvis))
- init prometheus server [\#10](https://github.com/babylonchain/babylon-relayer/pull/10) ([SebastianElvis](https://github.com/SebastianElvis))
- fix: automatically calculate TrustingPeriod and override previous light clients [\#9](https://github.com/babylonchain/babylon-relayer/pull/9) ([SebastianElvis](https://github.com/SebastianElvis))
- chore: flag for number of retry attempts [\#8](https://github.com/babylonchain/babylon-relayer/pull/8) ([SebastianElvis](https://github.com/SebastianElvis))
- feat: create light client if it does not exist on Babylon [\#7](https://github.com/babylonchain/babylon-relayer/pull/7) ([SebastianElvis](https://github.com/SebastianElvis))
- docker: Dockerfile for Babylon relayer [\#6](https://github.com/babylonchain/babylon-relayer/pull/6) ([SebastianElvis](https://github.com/SebastianElvis))

## [v0.1.0](https://github.com/babylonchain/babylon-relayer/tree/v0.1.0) (2023-01-04)

[Full Changelog](https://github.com/babylonchain/babylon-relayer/compare/da4f49693189046a7bfeab192cd0cde8868595e7...v0.1.0)

**Closed issues:**

- Use the latest commit of the Go relayer as dependency [\#3](https://github.com/babylonchain/babylon-relayer/issues/3)

**Merged pull requests:**

- feat: multiplexed relayer [\#5](https://github.com/babylonchain/babylon-relayer/pull/5) ([SebastianElvis](https://github.com/SebastianElvis))
- feat: extra codec support and minor improvements [\#4](https://github.com/babylonchain/babylon-relayer/pull/4) ([SebastianElvis](https://github.com/SebastianElvis))
- First version of Babylon relayer implementation [\#2](https://github.com/babylonchain/babylon-relayer/pull/2) ([SebastianElvis](https://github.com/SebastianElvis))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
