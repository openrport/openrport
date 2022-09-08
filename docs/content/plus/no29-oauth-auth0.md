---
title: "OAuth > Auth0"
weight: 29
slug: oauth-auth0
---

## Overview

To use Auth0 users for Rport authentication you must add and configure an xxx. This App is
created and fully controlled by the Rport adminstrator and permission for Rport to use the resulting access and
id tokens can be ??? at any time. When users first login to Rport via their Auth0 user, they must
allow this app to read their xxx info.

By specifying a `required_role` in the rportd config, user access to Rport can be limited solely to
the members who have that Auth0 role.

If the `permitted_user_list` config option is not set or set to `false`, then rportd will automatically
create and add any user who successfully authenticates with Auth0 to the list of allowed users for Rport.
Note the `required_role` config param must be set for this to apply.

## Setup

Steps

1. ...

11. For configuring the rportd server config file, the following information will be required:

  ```toml
  provider = "auth0"
  authorize_url = ""
  token_url = ""
  redirect_uri = "<xxx (from step 7)"
  client_id = "your app/client id"
  client_secret = "your client secret (from step xx)"
  ```

12. Set the rportd oauth access control config parameters as required

tbd

### Auth0 and Rport Usernames

tbd
