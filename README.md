# Webshell

This Golang application establishes a WebSocket connection to handle requests
from xterm.js clients. It allows multiple clients to run commands through PTY
(pseudo-terminal), providing a real-time terminal interface over the web without
the need to open any additional ports. The WebSocket router manages the
connections and routes the requests appropriately.

## Why?

The primary purpose of this Golang application is to facilitate debugging and
issue triage across multiple VMs or clusters through a web UI, without needing
to authenticate with each individual service. Here’s a more detailed
explanation:

- **Centralized Access:** Streamlines the process of managing and debugging
  issues across a distributed infrastructure.
- **Authentication:** By connecting to a centralized WebSocket server, users can access multiple
  environments without dealing with the complexities of managing multiple
  credentials.
- **Audit Logging:** Users authenticate once with the WebSocket server, which then
  manages the connections to the individual VMs or clusters. This centeralize
  point can be used for audit logs.

### Use Case Scenarios

- **Multi-Cluster Management:** Easily manage and debug issues across multiple
  Kubernetes clusters without repeatedly authenticating with each cluster.
- **Remote Server Access:** Provide remote access to various VMs for system
  administrators, allowing them to troubleshoot and resolve issues from a single
  web UI.
- **DevOps Support:** Enhance DevOps workflows by providing a centralized tool
  for accessing different environments, making it easier to deploy, monitor, and
  debug applications.
