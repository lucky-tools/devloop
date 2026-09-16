---
title: "Privacy Settings"
linkTitle: "Privacy Settings"
weight: 50
---

The privacy of our users is very important to us. 
Your use of this software is subject to the <a href=https://policies.google.com/privacy>Google Privacy Policy</a>.

## Update check

To keep Devloop up to date, update checks are made to Google servers to see if a new version of
Devloop is available. By default, this behavior is enabled. As a side effect this request is logged.
 
To disable the update check you have two options:

1. set the `DEVLOOP_UPDATE_CHECK` environment variable to `false`
2. turn it off in devloop's global config with: 
```bash
    devloop config set -g update-check false
```
