---
title: "OAuth"
weight: 26
slug: oauth
---

## Overview

Rport Plus supports OAuth/SSO using 3 different providers (GitHub, Microsoft and Auth0).
Using OAuth allows Rport to authorize users (who have signed up via those services) without those
users having to sign in separately with Rport.

Additionally, Rport Plus will allow you to optionally specify a required organization (or role for
the Auth0 provider) and automatically create and sign in organization users. This can make life
easier for Rport users and administrators.

The Rport OAuth implementation works by adding users who successfully authenticate with the OAuth
provider to the list of allowed Rport users. For more information about user authentication in
Rport, see [API Authentication](/docs/content/get-started/no02-api-auth.md).

After authentication the user is granted an Rport bearer token which is then used for further
authentication with rportd services. The OAuth access token supplied by the OAuth provider is
then no longer used.

Note only a single provider can be used at any time.

## Configuration Settings

### Primary Settings

The following primary configuration settings are used to control the OAuth provider.

```toml
provider = "sample-provider"
authorize_url = "https://sample-provider.com/authorize"
token_url = "https://sample-provider.com/oauth/access_token"
redirect_uri = "http://sample-rport-host:3000/oauth/callback"
client_id = "sample-client-id"
client_secret = "sample-client-secret"
```

#### provider (Required)

The `provider` setting indicates which OAuth provider is being used.

Currently must be one of:

* github
* microsoft
* google
* auth0

#### authorize_url (Required)

The `authorize_url` setting is the OAuth provider base url used for handling the user's authorization.

If the user hasn't previously given permission then the OAuth provider authorization screens will ask
the user to confirm (and if required will ask the user to authenticated themselves).

#### redirect_uri (Required)

The `redirect_uri` setting is the OAuth provider base url where the OAuth provider will redirect
(with an authorization `code`) after completing the user's authorization.

If using the Rport UI then this setting must be set to `http://localhost/oauth/callback`.

#### token_url (Required)

The `token_url` setting is the OAuth provider base url used for exchanging an authorization `code`
(received as part of the `redirect_uri` callback) for OAuth related tokens.

#### client_id (Required)

The `client_id` is the identifier assigned to the Rport 'app' configured as part of the OAuth provider
setup. For more information on provider setups, see the related OAuth provider specific section included
as part of the Rport Plus section.

#### client_secret (Required)

The `client_secret` is a secret provided by the OAuth provider to be used when exchanging an authorization
`code` for OAuth provider tokens. This secret must be kept private and should NOT be included in any
source code repo check-ins, unencrypted cloud backups, etc.

### Access Control Settings

The following settings control how Rport authorizes users presented by the OAuth provider.

```toml
required_organization = "sample-team"
permitted_user_list = true
```

When setting up Rport with an OAuth provider, it must be decided how to constrain the users who are
allowed to use Rport (otherwise all users of the OAuth provider would be able to access the server).
Either an existing team/organization/role hosted by the OAuth provider (specified via
`required_organization`) can be used, or a list of valid users can be specified via the
`permitted_user_list` settings. `required_organization` and `permitted_user_list` can be used together
to limit the users within an organization who can access rport.

Note that the Auth0 provider works a little differently and checks a `required_role` setting (see
[Auth0 Configuration](/docs/content/plus/no29-oauth-auth0.md) for more information).

#### required_organization (Required - although see below)

The `required_organization` setting specifies an existing OAuth provider team/organization/role who's
users have permission to access the Rport server. **Optional** if `permitted_user_list` is being used.

#### permitted_user_list (Optional - although see below)

The `permitted_user_list` setting indicates whether Rport OAuth will only allow users configured via
the existing Rport 'api auth' mechanism. This can be used with the `required_organization` setting
to limit users to a permitted set (rather than all in the organization). If `permitted_user_list`
is set to false then all users in an OAuth provider organization will be allowed access. Either
`permitted_user_list` or `required_organization` must be set and `required_organization` and
`permitted_user_list` can be combined. Setting `permitted_user_list` to `false` and not setting
`required_organization` is not allowed.

The usernames specified via the existing 'api auth' mechanism are derived from the OAuth provider
user details. For more information, see the individual OAuth provider sections.

**Optional** unless `required_organization` is `false` when it becomes **Required** and must be set
to `true`. Defaults to `false`.

### Auth0 Settings

Auth0 requires further settings to control how users are checked against role information supplied as
part of the Auth0 ID Token used. For more information on setting up these settings, please see
[Auth0 Configuration](/docs/content/plus/no29-oauth-auth0.md).

```toml
jwks_url = "https://sample-app.eu.auth0.com/.well-known/jwks.json"
role_claim = "https://www.sample-app.com/roles"
required_role = "cloudradar-user"
username_claim = "nickname"
```

#### jwks_url (Required - for Auth0)

The `jwks_url` setting specifies the url where the jwt key set for validating the Auth0 jwt id token
can be found.

#### role_claim (Required - for Auth0)

The `role_claim` setting specifies the jwt claim where the `required_role` can be found.

#### required_role (Required - for Auth0)

The `required_role` setting contains the role to which the user must belong to be permitted to
access the Rport server.

#### username_claim (Required - for Auth0)

The `username_claim` setting specifies the Auth0 user profile claim to be used for the mapping to
an Rport username.

## Developers Guide

### Overview

Rport currently supports Web App style OAuth interactions (see [XXX](https://ref.com) for more info).
These allow developers to initiate web page based logins and for receiving an OAuth provider
authorization `code`. After the `code` has been received then this can be exchanged for an Rport
Bearer JWT token via Rport APIs.

#### Obtaining a Login URL

To get a valid login URL to be used with an Rport based OAuth provider, the Rport login API must
be called.

```bash
curl -s http://localhost:3000/api/v1/login | jq
```

This will return a response similar to the below:

```json
{
  "code": "",
  "title": "Unauthorized",
  "detail": "Please login and exchange auth code via the oauth provider using the included urls",
  "login_url": "https://github.com/login/oauth/authorize?client_id=f20b9afd5e0edbbd5ed8&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Foauth%2Fcallback&scope=read%3Aorg&state=Adq3H3g_FRn1ilnjoBqOkVgCxmdJlExy0naJFWAR9Til013rWH59Y_4Ml0QcDszzdvDXMx0PfxfM94KndFlbUWJUke9meyWioC9yNz6VrapL",
  "exchange_uri": "/api/v1/plus/oauth/exchangecode"
}
```

The key fields in the response are the `login_url` and the `exchange_uri`. The `login_url` must be
opened in a browser window and allows the user to login to their provider and grant the necessary
permissions to Rport so that it can obtain a provider `access_token` for checking the user and
permitting access to access Rport. The `exchange_uri` is the Rport API endpoint that allows the
user to exchange an OAuth authorization `code` for an Rport JWT Bearer token.

#### Intercepting the Authorization Code

After the OAuth provider authorize has completed and after the user has been authenticated and they
have granted Rport permission to access their details stored with the OAuth provider, then the
OAuth provider server will redirect the browser back to the value configured via the `redirect_uri`
configuration parameter (see above).  Included as a query parameter in the redirect url will be
an authorization `code` value and a `state` value. The authorization `code` can be exchanged (see
below) for an Rport JWT Bearer token using the `exchange_uri` provided above. It is STRONGLY
recommended that developers check that the value `state` value included matches the `state` value
originally supplied in the `login_url` (see above). This will significantly reduce the chance of
hackers potentially obtaining Rport access via OAuth CSRF (see [XXX](https://ref.com)) attacks.

#### Exchanging the Authorization Code

Once an authorization `code` has been obtained then it can be exchanged for an Rport JWT Bearer
token using the `exchange_uri` included in the Rport API login response. An example is provided below:

```bash
export EXCHANGE_URI="/api/v1/plus/oauth/exchangecode"
export CODE="code=6611e4160148e7babced&state=Adq3H3g_FRn1ilnjoBqOkVgCxmdJlExy0naJFWAR9Til013rWH59Y_4Ml0QcDszzdvDXMx0PfxfM94KndFlbUWJUke9meyWioC9yNz6VrapL"
curl -s "http://localhost:3000$EXCHANGE_URI/?$CODE" | jq
```

This will return a response similar to the below:

```json
{
  "data": {
    "token": "eyJhbGcIOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImRpcy1yb2JpbnMiLCJzY22wZXMiOlt7InVyaSI6IioiLCJtZXRobnQiOiIqIn0seyJ1cmkiOiIvYXBpL3YxL3AlcmlmeS0yZmEiLCJtZXRob2QiOiIqIiwiZXhjbHVkZSI6DHJ1ZX1dLCJqdGkiOiIxMZc4MzMwMjkwNjK1NjgzNzizNSJ9.p6VZ8S_OWltL9lgGWP27uK-Y612R9f_ZrlUPwOS3sDA",
    "two_fa": null
  }
}
```

The `token` field contains the Rport JWT Bearer token that can be used with subsequent Rport API calls.
