---
name: Bug report
about: Create a report to help us improve
title: ''
labels: ''
assignees: ''

---

**Describe the bug**
A clear and concise description of what the bug is.

A feature request is not a bug. Feature requests are welcome. Follow [this link](https://github.com/cloudradar-monitoring/rport/discussions/categories/roadmap-feature-wishes-ideas).

In case you know it. Is the bug related to `rportd` (server) or `rport` (client)?

**Environment**
Describe the environment where the bug occurs:
* RPort version
* OS details (Linux distribution, and version or Windows version)
* Is rportd running inside a docker container?
* Are you using a reverse proxy? If yes, which one?

**Log file**
Before submitting a bug report, inspect the log files of client and server.
`/var/log/rport/rportd.log` for the server and `/var/log/rport/rport.log` or `C:\Program Files\rport\rport.log` for the client.
[Increase the log level to debug](https://github.com/cloudradar-monitoring/rport/blob/0.9.0/rport.example.conf#L169-L171) and try to reproduce the error. 
Include relevant lines into your report.

**Screenshots**
If applicable, add screenshots to help explain your problem.

**Additional context**
Add any other context about the problem here.
