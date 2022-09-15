---
title: "OAuth > GitHub"
weight: 27
slug: oauth-github
---

## Overview

To use GitHub users for Rport authentication you must add and configure a GitHub OAuth App.
This App is created and fully controlled by the Rport administrator and permission for Rport access
can be revoked at any time. When users first login to Rport via their GitHub user, they must
allow this app to read their profile and org info.

By specifying a `required_organization` in the rportd config (see the [OAuth](/docs/content/plus/no26-oauth.md)
section for more information), user access to Rport can be limited solely to the members of an
individual GitHub organization without needing to setup the users in advance (see below).

If the `permitted_user_list` config option is not set or set to `false`, then rportd will automatically
create and add any user who successfully authenticates with GitHub to the list of allowed users for
Rport. Note the `required_organization` config param must be set for this to apply.

## Setup

Steps

1. Login to the GitHub account that you wish to use as the admin for the Rport Access App

2. Select *Settings* from the top-right GitHub menu

3. Select *Developer Settings* from the bottom of left side-bar menu

4. Select *OAuth Apps* from the *Developer Settings* left side-bar menu

5. Click *Register a new application*

6. Enter the details requested on the *Register a new OAuth application* screen (see 7 below)

7. Enter the *Authorization callback URL*. Most likely this needs to be `http://localhost/oauth/callback`,
so that clients can intercept the returned authorization `code`. If using the Rport UI
then this value MUST match the `localhost` setting.

8. Click *Register application*

9. Review the details presented for the newly created app. Note the `client id`.

10. In the section titled *Client secrets*, click *Generate a new client secret*. Copy and paste the
generated secret and keep somewhere safe.

11. For configuring the rportd server config file, the following information will be required:

  ```toml
  provider = "github"
  authorize_url = "https://github.com/login/oauth/authorize"
  token_url = "https://github.com/login/oauth/access_token"
  redirect_uri = "<your authorization callback URL (from step 7)>"
  client_id = "<your client id> <from step 9>"
  client_secret = "<your client secret (from step 10)>"
  ```

12. Set the rportd oauth access control config parameters as required

Depending on requirements, the following access control config parameters maybe set.

  ```toml
  # Users must be members of the cloudradar.io organization
  required_organization="cloudradar-monitoring"
  # All members are permitted to access Rport
  permitted_user_list=false
  ```

Note: The required_organization must match the organization name as displayed in the
GitHub URL for the organization or as under the list of organizations for which
the user is a member. For example:

```bash
https://github.com/cloudradar-monitoring
```

The organization name that must match the required_organization is `cloudradar-monitoring`.

13. Restart the rportd server with the configuration settings set as above.

### Github and Rport Usernames

Rport OAuth for GitHub uses the `login` field from the GitHub API user details as the username for
Rport. Please see [Get the authenticated user](https://docs.github.com/en/rest/users/users#get-the-authenticated-user) for a description of this field.

### Checking the Required Organization

For the required organization check, Rport checks that the `required_organization` configuration
value is one of the `login` values for the orgs of which the GitHub user is a member. Please see
GitHub REST [List organizations for the authenticated user](https://docs.github.com/en/rest/orgs/orgs#list-organizations-for-the-authenticated-user) API for
more information. Note that Rport will only check the first 100 orgs that a user belongs to.
