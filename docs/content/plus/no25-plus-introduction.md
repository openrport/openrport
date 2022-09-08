---
title: "Introduction"
weight: 25
slug: intro
---

## Overview

Rport Plus extends the functionality of Rport via a paid non open-source plugin. The plugin adds
capabilities that can be individually enabled and configured.

The first release of Rport Plus initially supports a single additional capability for SSO/OAuth via
one of either GitHub, Microsoft, Google or Auth0.

## Configuration

To use the Rport Plus functionality, the Rport Plus plugin must be loaded by Rport and the location
of the plugin must be specified in the rportd configuration file. The `[plus-plugin]` section and
`plugin_path` option must both be set.

```toml
[plus-plugin]
plugin_path = "./rport-plus.so"
```

For more information on configuring individual capabilities (such as SSO/OAuth), see the relevant
sections of the Rport Plus documentation.
