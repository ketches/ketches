# Ketches

English | [简体中文](./README_zh-CN.md)

A cloud-native application platform for building and deploying applications.

## Installation

### Kubernetes

1. Make sure you have a Kubernetes cluster running and `kubectl` configured to access it.
2. Apply the Kubernetes manifests to deploy Ketches:

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/ketches/master/deploy/kubernetes/manifests.yaml
```

### Docker Compose

1. Make sure you have Docker and Docker Compose installed.
2. Download the Docker Compose file:

```bash
mkdir ketches && cd ketches
curl -O https://raw.githubusercontent.com/ketches/ketches/master/deploy/docker-compose/docker-compose.yaml

docker compose up -d
```

### Source Code

1. Clone the repository:

```bash
git clone https://github.com/ketches/ketches.git
cd ketches
```

2. Run the backend server(Storage is SQLite by default, you can change it in the `.env` file):

```bash
cd backend
make run
```

This will migrate the database automatically and start the server on port 8080.

### Frontend

1. Make sure environment variables set in `[.env](./frontend/.env)` file are correct.
2. Run the frontend server

```bash
cd frontend
yarn
yarn dev
```

This will start the frontend server on port 5173.

Here you go! You can now access the Ketches application at `http://localhost:5173`.

## Backend Congiguration

T
Moe details about backend configurations, see [docs/backend-env-en.md](./docs/backend-env-en.md).

## Features

Admin panel

---

- [x] User management
  - [x] User sign-up
  - [x] User sign-in
  - [x] User sign-out
  - [ ] User profile management
- [ ] Cluster management(WIP)
  - [x] Add cluster in KubeConfig format
- [ ] Cluster extension management(WIP: Observability, Gateway-API, AI-Analytics)
- [x] Multi-cluster management

User panel

---

- [ ] Project management(WIP)
  - [x] Project membership management
- [ ] Env management(WIP)
- [ ] App management(WIP)
  - [x] Deploy app in container image format
  - [ ] Deploy app in kubernetes manifest format
  - [ ] Deploy app in source code format
  - [ ] Deploy app from AppHub
  - [x] App environment variables management
  - [x] App volume management
  - [ ] App mutli-container management(Plugins)
  - [ ] App gateway management
  - [ ] App health check management
  - [ ] App scaling management
  - [ ] App schedule management
  - [x] App instance container logs
  - [x] App instance container terminal
  - [ ] App observability(Need cluster extension installed)
  - [ ] App logs archive(Need cluster extension installed)
- [ ] Volume management(WIP)
- [ ] AppHub management(WIP)

- [ ] ...

## Screenshots

![alt text](docs/images/app-page.png)
![alt text](docs/images/app-instance-logs.png)
![alt text](docs/images/app-instance-terminal.png)
