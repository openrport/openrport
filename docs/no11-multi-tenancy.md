# Multi tenancy (NOT SUPPORTED YET)
Rport server can isolate clients and users from different tenants.
Multi tenancy is only supported when a database table is used for client- and api authentication.
`auth_table`, `auth_user_table`, and `auth_group_table` must be used.
Enabling multi_tenancy = true without the above prerequisites causes the rport server to exit with an error.

If multi tenancy is enabled the user auth table and the client auth table need an additional column `tenant` either varchar or int, ideally with an index.
Usernames must be unique across tenants, otherwise mapping users to a tenant would fail.

## Examples

:::: code-group
::: code-group-item MySQL
```mysql
CREATE TABLE `users` (
  `username` varchar(150) NOT NULL,
  `password` varchar(255) NOT NULL,
  `tenant` varchar(50) NOT NULL,
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
CREATE TABLE `groups` (
  `username` varchar(150) NOT NULL,
  `group` varchar(150) NOT NULL,
  `tenant` varchar(50) NOT NULL,
  UNIQUE KEY `username_group_tenant` (`username`,`group`,`tenant`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```
:::
::: code-group-item SQlite3
```sqlite
CREATE TABLE "users" (
  "username" TEXT(150) NOT NULL,
  "password" TEXT(255) NOT NULL,
  "tenant" TEXT(50) NOT NULL
);
CREATE UNIQUE INDEX "main"."username"
ON "users" (
  "username" ASC
);
CREATE TABLE "groups" (
  "username" TEXT(150) NOT NULL,
  "group" TEXT(150) NOT NULL,
  "tenant" TEXT(50) NOT NULL
);
CREATE UNIQUE INDEX "main"."username_group_tenant"
ON "groups" (
  "username" ASC,
  "group" ASC,
  "teant" ASC
);
```
:::
::::

## How it works
Having multi-tenancy enabled each rport client is mapped to his tenant on connect. The client tables show the additional tenant object.
```
curl -s -u admin:foobaz http://localhost:3000/api/v1/clients|jq
[
  {
    "tenant": "Doe",
    "id": "2ba9174e-640e-4694-ad35-34a2d6f3986b",
    "name": "My Test VM",
    "os": "Linux my-devvm-v3 5.4.0-37-generic #41-Ubuntu SMP Wed Jun 3 18:57:02 UTC 2020 x86_64 x86_64 x86_64 GNU/Linux",
    "os_arch": "amd64",
    "os_family": "debian",
    "os_kernel": "linux",
    "os_full_name": "Debian",
    "os_version": "5.4.0-37",
    "os_virtualization_system":"KVM",
    "os_virtualization_role":"guest",
    "cpu_family":"59",
    "cpu_model":"6",
    "cpu_model_name":"Intel(R) Xeon(R) Silver 4110 CPU @ 2.10GHz",
    "cpu_vendor":"Intel",
    "num_cpus":16,
    "mem_total":67020316672,
    "timezone":"UTC (UTC+00:00)",
    "hostname": "my-devvm-v3",
    "ipv4": [
      "192.168.3.148"
    ],
    "ipv6": [
      "fe80::20c:29ff:fec8:b1f"
    ],
    "tags": [
      "Linux",
      "Office Berlin"
    ],
    "version": "0.1.6",
    "address": "87.123.136.***:63552",
    "tunnels": []
  }
]
```
API users will only see clients having the tenant mapped to the user. A user without a tenant will always get an empty list.
A client without a tenant will be orphaned and will not appear in any client listing.
It's good to make sure on database-level the tenant column cannot be null.
