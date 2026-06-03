# NEXUS Licensing

NEXUS is built to be useful, trustworthy and hard to clone. This page explains
exactly what that means today and how it can change later.

## Today — Apache-2.0 (Community edition)

Every NEXUS release through v0.5.x ships under the **Apache License, Version
2.0** (see [`LICENSE`](../LICENSE)). The full license text lives in the
repository root and is shipped inside every binary's source tree.

Apache-2.0 was chosen over MIT for two concrete reasons:

1. **Trademark protection.** Section 6 of Apache-2.0 forbids the use of the
   licensor's trademarks. Combined with the trademark notice in
   [`NOTICE`](../NOTICE), this means a fork is free to take the *code*, but it
   is **not** free to redistribute it under the "NEXUS" name, the "nexus-proxy"
   name, or a confusingly similar name. That blocks the most common kind of
   "clone": a renamed-but-otherwise-identical hosted service.
2. **Patent grant + retaliation.** Apache-2.0 grants users a patent license and
   automatically terminates that license if they sue NEXUS contributors over
   the same patents. MIT has neither.

You may freely use, modify, run, embed, sell, and redistribute NEXUS today,
subject to those two protections.

## A licensing layer in the code

The Go package [`internal/license`](../internal/license/) is the runtime seam
that lets a future build behave differently per edition or per feature without
touching every call site:

```go
l := license.Active()                  // *License, never nil
if l.Allow(license.FeatureCascade) { … } // today: always true
```

Today every build returns the same `Community` edition with every feature
unlocked. `nexus version` surfaces the active license:

```
nexus v0.5.0 (built …)
license: Community edition · Apache-2.0 · all features unlocked (community)
```

The reserved env var `NEXUS_LICENSE_KEY` is recognised but explicitly a no-op
today — see the tests in `internal/license/license_test.go`.

## Future — source-available for newer versions (optional)

If at some point we want to keep newer versions out of the hands of
clone-and-host operators, NEXUS can switch **future releases** to the
**Business Source License 1.1** (BSL). BSL is "source-available, not
production-use without a commercial license, but auto-converts to a real OSS
license after a fixed Change Date." It is the same license HashiCorp, Sentry,
MariaDB and CockroachDB use.

The template below is filled in for NEXUS and ready to drop in as `LICENSE`
when (and only when) the project wants to make the switch. Existing released
versions (v0.5.x and earlier) **stay Apache-2.0 forever** — BSL would apply
only to versions tagged after the switch.

```
Business Source License 1.1

Licensor:             NEXUS contributors
Licensed Work:        NEXUS (the version tagged at or after the change below)
Additional Use Grant: You may make production use of the Licensed Work,
                      provided that your use does not include offering the
                      Licensed Work to third parties on a hosted or embedded
                      basis as a competing managed proxy, LLM gateway, or
                      cost-routing service.
Change Date:          Four years from the date of the first publicly available
                      distribution of a specific version of the Licensed Work
                      under this License.
Change License:       Apache License, Version 2.0
```

(The official Business Source License 1.1 body text follows the
[canonical version](https://mariadb.com/bsl11/). Do not modify it — only the
four parameters above.)

### What BSL would and would not change

| | Today (Apache-2.0) | Future (BSL 1.1) |
|---|---|---|
| Read, fork, modify the source | ✅ | ✅ |
| Self-host on your own machines | ✅ | ✅ |
| Internal company use | ✅ | ✅ |
| Use the name "NEXUS" on a fork/clone | ❌ | ❌ |
| Sell a hosted "NEXUS as a service" | ✅ (but blocked by trademark) | ❌ until Change Date |
| OSI-approved "open source" | ✅ | ❌ (source-available) |

## Contributing under this licensing model

Contributions are accepted under the **inbound = outbound** rule: by opening a
pull request you agree that your contribution is licensed under the same terms
as the file you are modifying (today: Apache-2.0). This is the same model the
Linux kernel and most large Apache-2.0 projects use, and it keeps the door open
to a future BSL switch without needing to chase every past contributor for
permission.

If you intend to relicense a substantial contribution differently, say so in
the PR description and we'll discuss before merging.

## Trademark policy (short version)

* "NEXUS" and the NEXUS logo are unregistered trademarks of the NEXUS project
  maintainers.
* You may **describe** your project as "compatible with NEXUS" or "powered by
  NEXUS" if it really is.
* You may **not** distribute a fork, hosted service, or commercial product
  *as* "NEXUS", "nexus-proxy", or a confusingly similar name without prior
  written permission.

See [`NOTICE`](../NOTICE) for the full version.
