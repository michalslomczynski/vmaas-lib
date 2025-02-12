# Changelog

<!--next-version-placeholder-->

## v0.3.5 (2023-03-20)
### Fix
* Re-use logging logic from patchman ([`e5af24b`](https://github.com/RedHatInsights/vmaas-lib/commit/e5af24b2ec71e6aa86c97a7f60620d58155aa37f))

## v0.3.4 (2023-03-20)
### Fix
* Stream downloaded dump to a file ([`0f49948`](https://github.com/RedHatInsights/vmaas-lib/commit/0f4994885461be647548c0cd137132cd88540803))

## v0.3.3 (2023-02-07)
### Fix
* Third_party json field ([`c991822`](https://github.com/RedHatInsights/vmaas-lib/commit/c991822e1aba9a6be8a4f290e089b8d07fdd76ba))

## v0.3.2 (2023-02-06)
### Fix
* Return errata: [] instead of null ([`9549f8a`](https://github.com/RedHatInsights/vmaas-lib/commit/9549f8a5d1ef160e942d6557b79ede50ac6d4c95))

## v0.3.1 (2023-01-19)
### Fix
* Allow nil repolist ([`96f4b79`](https://github.com/RedHatInsights/vmaas-lib/commit/96f4b79c8efff5095fb0059d8fa2423d9d5377c8))

## v0.3.0 (2023-01-11)
### Feature
* Add goroutines ([`7eb7548`](https://github.com/RedHatInsights/vmaas-lib/commit/7eb754806bc4885df09b49d4d1b5563822d1d065))

## v0.2.6 (2023-01-05)
### Fix
* Detail load, unnecessary cve iteration ([`a83a6e6`](https://github.com/RedHatInsights/vmaas-lib/commit/a83a6e6895e3a21666c9169e29bf8c369baacc08))

## v0.2.5 (2023-01-04)
### Fix
* Cache reload ([`9a8a676`](https://github.com/RedHatInsights/vmaas-lib/commit/9a8a676485444ce77c3e4d9c2bdae62f343f88c0))

## v0.2.4 (2022-12-16)
### Fix
* Pre-alloc maps in cache ([`8f4eba6`](https://github.com/RedHatInsights/vmaas-lib/commit/8f4eba6dc2b45fea0b09b07c3f9a9d4f5a196cb7))

## v0.2.3 (2022-12-14)
### Fix
* Use nevra pointer for receiver ([`e0d8a9f`](https://github.com/RedHatInsights/vmaas-lib/commit/e0d8a9f00970cf12720f3eb1d979a3d09bdada55))
* Close db after cache read ([`a9486e3`](https://github.com/RedHatInsights/vmaas-lib/commit/a9486e36ff8a31d5810c68511fb6b4453053e376))
* Optimize oval load ([`b6d7e01`](https://github.com/RedHatInsights/vmaas-lib/commit/b6d7e01ddc98e4d346ed4f8c58941252a8a25738))
* Reduce number of allocations ([`38d1be5`](https://github.com/RedHatInsights/vmaas-lib/commit/38d1be54de528b014ce8a9c1c3f30a8a8f5a3258))

## v0.2.2 (2022-12-09)
### Fix
* Updates when releasever in repo is empty ([`3ec8712`](https://github.com/RedHatInsights/vmaas-lib/commit/3ec8712cdaa5638902ee1d2b6aecf31b3c3de0a8))

## v0.2.1 (2022-12-08)
### Fix
* Arch compatibility ([`b18e816`](https://github.com/RedHatInsights/vmaas-lib/commit/b18e816f253edd3dcd580aaf5854024c7b9b3e7d))

## v0.2.0 (2022-12-08)
### Feature
* **rhui:** Look up updates by repository path ([`044abab`](https://github.com/RedHatInsights/vmaas-lib/commit/044abab43674b1836874cd172ce3187293b57b80))

## v0.1.4 (2022-12-01)
### Fix
* Minor fixes ([`9c06686`](https://github.com/RedHatInsights/vmaas-lib/commit/9c06686039c1386efd8948a1cef91da4e7267766))

## v0.1.3 (2022-11-30)
### Fix
* Issues found with unit tests ([`43beb51`](https://github.com/RedHatInsights/vmaas-lib/commit/43beb5188c98c0d0edbdf6816fe358d516c6cdbb))

## v0.1.2 (2022-11-28)
### Fix
* Don't iter UpdatesIndex in processInputPackages ([`8f2fc92`](https://github.com/RedHatInsights/vmaas-lib/commit/8f2fc92a1d39ea8ffcd59e9b77038fa1afbd571e))

## v0.1.1 (2022-11-28)
### Fix
* RepoID slice, simplify intersection, gorpm build ([`1611883`](https://github.com/RedHatInsights/vmaas-lib/commit/1611883bebc2856c232b8385200990d21d1b83c3))

## v0.1.0 (2022-11-28)
### Feature
* **test:** Introduce unit tests ([`27584fb`](https://github.com/RedHatInsights/vmaas-lib/commit/27584fba178ccf2c3bc34b6ceb7708dd74859e49))
* Setup semantic release from vuln4shift ([`01ccb51`](https://github.com/RedHatInsights/vmaas-lib/commit/01ccb51313e4a520f3a8bb9d4e06955ec1e95fe0))
