---
title: "OAuth > Microsoft"
weight: 28
slug: oauth-microsoft
---

## Overview

To use Microsoft users for Rport authentication you must add and configure an *App registration* for
Rport. This App is created and fully controlled by the Rport administrator and permission for Rport
access can be revoked at any time. When users first login to Rport via their Microsoft user, they
must allow this app to read their user info.

By specifying a `required_organization` in the rportd config, user access to Rport can be limited
solely to the members of a Microoft adminstrative unit / group / role.

If the `permitted_user_list` config option is not set or set to `false`, then rportd will
automatically create and add any user who successfully authenticates with Microsoft to the list of
allowed users for Rport. This means users do not need to be setup in advance. Note the
`required_organization` config param must be set for this to apply.

## Setup

Steps

1. Create a Microsoft Azure Account and Subscription

2. Make sure you are on the Azure portal home page `https://portal.azure.com/#home`

3. Select *Azure Active Directory* from the *Azure services* section

4. Select *App registrations* from the left side-bar

5. Select *New registration* from the left-side of the top menu-bar

6. Complete the details in the *Register an application* form. Most likely you'll want to select the
*Single tenant* option for *Supported account types* and *Web* for the Redirect URI

7. Click the *Register* button to register the app

8. Review the presented details on the screen displayed (titled with your app display name). Note
the *Application (client) ID* and the *Directory (tenant) ID*.

9. Click the *Add a certificate or secret link* on the right-side of the *Essentials* section.

10. Make sure the middle option *Client secrets* is selected and click the *+ New client secret* option
at the top of the section.

11. Enter a *description* and *Expire duration* and click *Add*

12. Ensure to copy the `Value` (and not the `Secret ID`) for the newly created client secret

13. Click on the *Overview* option at the top of the left side-bar to return to the *App registration* summary

14. Click on *Endpoints* in the middle of the top menu-bar

15. Make a note of the *OAuth 2.0 authorization endpoint (v2)* and *OAuth 2.0 token endpoint (v2)* endpoints

16. For configuring the rportd server config file, the following information will be required:

  ```toml
  provider = "microsoft"
  authorize_url = "<the OAuth 2.0 authorization endpoint (from step 15)>"
  token_url = "<the OAuth 2.0 token endpoint (from step 15)>"
  redirect_uri = "<your redirect uri (from step 6)>"
  client_id = "your application (client) id (from step 8)>"
  client_secret = "your client secret (from step 12)"
  ```

17. Set the rportd oauth access control config parameters as required

Depending on requirements, the following access control config parameters maybe set.

  ```toml
  # Microsoft Azure ID of required adminstrative unit / group / role
  required_organization="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  ## allow all users within org unit
  permitted_user_list=false
  ```

18. Restart the rportd server with the configuration settings set as above.

### Microsoft and Rport Usernames

For the Rport `username`, Rport uses the `displayName` field from the Microsoft Graph REST
['user resource type - properties'](https://docs.microsoft.com/en-us/graph/api/resources/user?view=graph-rest-1.0#properties) API.

### Checking the Required Organization

For the required organization check, Rport checks that the `required_organization` configuration
value is one of the `id` values for which the Microsoft user is a `memberOf`. Please see Microsoft
Graph REST [List a user's direct memberships](https://docs.microsoft.com/en-us/graph/api/user-list-memberof?view=graph-rest-1.0&tabs=http)
API for more information. Note that the user must be a direct member of the relevant group/organization/role
and not a transitive member.
