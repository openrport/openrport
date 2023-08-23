---
title: "Fine-tuning the Rport Server: Understanding `max_concurrent_ssh_handshake`"
weight: 30
slug: "fine-tuing-rport-server"
---
{{< toc >}}

The `max_concurrent_ssh_handshake` parameter is a pivotal configuration detail in the rport server.
Here, we delve deep into what it represents and its implications, especially when dealing with a large client base.

---

## What is `max_concurrent_ssh_handshake`?

The `max_concurrent_ssh_handshake` parameter in the rport server was introduced as a defensive measure against
the "thundering herd" effect. This effect becomes notably pronounced post server downtimes,
when a deluge of clients try to reestablish their SSH connections simultaneously.
Handling numerous connection attempts and the extensive data associated with each handshake
can heavily tax both the CPU and network bandwidth, frequently leading to subsequent connection timeouts.
Through capping concurrent handshakes,
this parameter aspires to optimize server resource allocation and guarantee a more streamlined reconnection process.

---

## Addressing User Queries

### 1. Can this be set to "No limit" for scalable environments?

Setting this parameter to "No limit" isn't viable. Without a stipulated cap,
the server risks undue strain, notably during peak reconnection periods.

### 2. What are the effects of raising this parameter?

A loftier `max_concurrent_ssh_handshake` value allows a greater number of simultaneous SSH handshake processes.
However, it's crucial to remain wary of potential server resource saturation, particularly during mass reconnections.

### 3. How should we scale?

- **Infrastructure:** Ensure your server resources
 — CPU, memory, and network — are in sync with expected peak loads, especially post-downtime.

- **Client Strategy:** Pre-configured clients come with a growing backoff mechanism,
 aiding the server during peak times. This feature should inform the server's scaling strategy.

- **Tuning:**
  - Modify the `max_concurrent_ssh_handshake` based on past data,
 anticipated load patterns, and server performance metrics post-downtimes.
  - **Binary Search Tuning**: Begin with the total client count and methodically halve the `max_concurrent_ssh_handshake` value until a stable configuration is pinpointed.
  - **Number of cores**: from our experimentation one of the limits was CPU and in this scenario
  we found the total number of cores divided by 2 to yield most stable results.

### 4. What happens if we set the baseline to 100?

Setting the `max_concurrent_ssh_handshake` to a value like 100 caps the server
to processing a maximum of 100 concurrent SSH handshakes.

However, there are associated cascading implications:

- **CPU utilization**: SSH handshakes are CPU intensive and increasing this value to 100 with only 2 slow cores will
 cause a situation in which all 100 handshakes compete for CPU time and take so long to process that they all timeout and server can't establish any connection.
 While on 256 core machine it would be conservative setting.

- **Connection Queueing**: A low threshold can result in many clients queueing up, leading to prolonged waits.

- **Client Timeouts**: Protracted waits can cause client-side timeouts,
 notably problematic if most clients are attempting simultaneous reconnections.

- **Exponential Backoff**: Given the client-side exponential backoff strategy,
 timeouts can lead to elongated durations before subsequent reconnection attempts.
 Consequently, a lower handshake limit can result in extended timeframes
 before all clients manage successful reconnections.

---

## Additional Recommendations

- **Planning for Downtimes:** Forewarn of upcoming server downtimes,
 encourage staggered client reconnections, or consider rolling restarts to alleviate the thundering herd effect.

- **Monitoring:** Monitor essential metrics like CPU usage, network bandwidth,
 and connection success ratios, with a keen eye on post-downtime scenarios.

- **Testing:** Create controlled test environments to simulate post-downtime connection dynamics,
 ensuring optimal live configurations.

- **Feedback Loop:** Create alert mechanisms for potential resource overconsumption instances,
 such as CPU spikes or bandwidth bottlenecks, ensuring timely interventions.
