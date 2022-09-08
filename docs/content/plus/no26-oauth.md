---
title: "OAuth"
weight: 26
slug: oauth
---

## Overview

Rport Plus supports OAuth/SSO using 4 different providers (GitHub, Microsoft, Google and Auth0).
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
