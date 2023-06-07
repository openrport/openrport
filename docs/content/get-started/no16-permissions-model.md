---
title: "Permissions Model"
weight: 16 
slug: "permissions-model"
---
{{< toc >}}

## Admin users

Members of the 'Administrators' user group bypass the below permission model completely. They have full access to all
hosts and all functions.

## User group permissions aka. function permissions

A none-admin user has no effective rights on rport unless via at least one user group, permissions are granted.
Currently, rport is subdivided into the following functions:

* tunnels
* scripts
* commands
* vault
* scheduler
* monitoring
* uploads
* auditlog

The permissions are stored on the `group_details` table of
your [API access database](/get-started/api-authentication/#database). They are managed through
the [update user groups API endpoint](https://apidoc.rport.io/master/#tag/User-Groups/operation/UserGroupPut).

In addition to one of the above function permissions client permissions are needed. In other words, the function
permissions define only what a user can do, but not on which clients he/she can do it.

{{< hint type=caution title="Permissions are additive only">}} There is no option to revoke a permission. Once a user
group has a permission, you cannot revoke it through a second user group. {{< /hint >}}

## Client permissions

A user that has at least one of the above function permission needs at least access to one client to effectively use
rport.

### Per client

You can grant access to a single client to one or many user groups. The allowed user groups are stored on the clients'
table inside the `details` object. The so-called client ACLs are managed through
the [client ACL API endpoint](https://apidoc.rport.io/master/#tag/Clients-and-Tunnels/operation/ClientAclPost)

### Client group permissions

You can grant access to a client group to one or many user groups. This makes managing access rights effective and
flexible.

Client group access is managed through
the [client groups API endpoint](https://apidoc.rport.io/master/#tag/Client-Groups). The underlying data is stored on
the `client_groups` database.

{{< hint type=caution title="Permissions are additive only">}} There is no option to revoke a permission. If permission
is granted to client group A, you cannot deny access to client group B if it is a subset of A.  
{{< /hint >}}

## Extended group permissions

With a valid RPort Plus license, you can grant access to a set of "extended permissions"
This feature enables Admins to narrow the permissions of a user group by restricting tunnels and commands.

As an example, one could limit a user group to only be able to create tunnels with a specific set of protocols,
or even to a specific set of hosts.

```json
"tunnels_restricted": {
    "scheme": ["ssh", "rdp"],
    "host_header": ":*",
}
```

In the same way one could limit the commands a user group can execute, defining regular expressions for allowed and /
or denied commands.

```json
"commands_restricted": { 
    "allow": ["^sudo reboot$","^systemctl .* restart$"], 
    "deny": ["^rm$","ssh"], 
} 
```

{{< hint type=caution title="License">}} Please note that using this feature requires a valid rport-plus license,
otherwise the extended permissions feature will run in trial mode.
{{< /hint >}}

Please refer to the [extended permissions documentation in RPort Plus](https://docs.rport.io/plus/extended-permissions/)
for more information.

### Trial Mode

To try out the extended permissions feature, you can use the trial mode.
This mode is enabled and its active by default, once the plugin is loaded, and database is upgraded with the extended
permissions feature.  
Running in trial mode, RPort Plus will validate a maximum of 5 Tunnels restrictions and 5 Commands restrictions, after
that, every validation will fail.

{{< hint type=tip title="Database upgrade">}}
table `group_details` must have columns `tunnels_restricted` and `commands_restricted` as described in the
[Database - Extended group permissions additional fields](/get-started/api-authentication/#extended-group-permissions-additional-fields)
page.

{{< /hint >}}

### User group permissions combined with extended permissions

When managing extended user group permissions, those permissions are combined with the general user group permissions,
as follows:

If a user group has both general permissions (e.g., "tunnels") and extended permissions (e.g., "tunnels_restricted"),
the wider permissions takes precedence. To effectively enable restricted tunnels or commands, the general correspondent
permissions must be granted.

If a user is part of multiple user groups, each with extended permissions, the permissions are combined. For example,
if one user group has a tunnel restriction of "ssh" and another has a tunnel restriction of "rdp", the user will be able
to create tunnels with either protocol.
