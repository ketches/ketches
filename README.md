# Ketches

A cloud-native application platform for building and deploying applications.

## Installation

### Backend

1. Configure configs in [config.yaml](./backend/config/config.yaml)

2. Run the backend server

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
  - [ ] App environment variables management
  - [ ] App volume management
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
