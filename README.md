# Yappy

[![Go](https://img.shields.io/badge/Language-Go-blue.svg )](https://golang.org/ )
[![License](https://img.shields.io/github/license/F0RG-2142/chirpy-proj )](LICENSE)

> A backend API mimicking some core features of X — built with Go, PostgreSQL, and JWT authentication. Designed as a portfolio project to demonstrate backend development skills.

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Technologies Used](#technologies-used)
- [API Endpoints](#api-endpoints)
- [Authentication Flow](#authentication-flow)
- [Database](#database)
- [Code Structure](#code-structure)
- [Security Practices](#security-practices)
- [License](#license)

---

## Overview

**Yappy** is a backend-only RESTful API that replicates key functionality of Twitter/X — including user registration, posting messages ("yaps"), deletion, updates, and authentication.

This project was built using:
- Go (with standard `net/http`)
- PostgreSQL for data persistence
- JWT-based authentication with access/refresh tokens
- Middleware patterns
- Structured JSON responses
- Profanity filtering

The code emphasizes clean architecture, separation of concerns, proper error handling, and secure practices such as password hashing and token management.

---

## Features

-  User registration and login
-  JWT access and refresh tokens
-  Token refresh and revocation
-  Posting and deleting yaps
-  Input validation and sanitization
-  Metrics tracking (request count)
-  Profanity filter (`LoL`, `fortnite`, `damn`)
-  Simulated premium upgrade via webhook

---

## Technologies Used

| Technology        | Purpose                                      |
|------------------|----------------------------------------------|
| Go               | The language  of choice for the project      |
| PostgreSQL       | Relational database                          |
| net/http         | Native HTTP server                           |
| jwt              | JSON Web Tokens for authentication           |
| bcrypt           | Password hashing                             |
| sqlc             | Type-safe DB interaction                     |
| goose            | Database migration tool                      |

---

## API Endpoints

| Method | Endpoint                                | Description                                  |
|--------|-----------------------------------------|----------------------------------------------|
| POST   | `/api/users`                            | Register a new user                          |
| POST   | `/api/login`                            | Log in an existing user                      |
| POST   | `/api/yaps`                             | Create a yap                                 |
| GET    | `/api/yaps/{authorId}`                  | Get all yaps by author                       |
| GET    | `/api/yaps/{yapId}`                     | Get a single yap                             |
| DELETE | `/api/yaps/{yapId}`                     | Delete a yap                                 |
| PUT    | `/api/users`                            | Update current user                          |
| POST   | `/api/refresh`                          | Refresh access token                         |
| POST   | `/api/revoke`                           | Revoke refresh token                         |
| GET    | `/admin/metrics`                        | View total request count                     |
| POST   | `/api/payment_platform/webhooks`        | Simulate premium upgrade (via webhook)       |

---

## Authentication Flow

1. **Register** a new user with `/api/users`
2. **Login** with `/api/login` → returns JWT and refresh token
3. Use the **JWT** in the Authorization header (`Bearer <token>`) for protected endpoints
4. When JWT expires, use `/api/refresh` with the refresh token to get a new one
5. Use `/api/revoke` to invalidate refresh token when logging out

All tokens are stored securely in the PostgreSQL database and are validated on each request.

---

## Database

This project uses **PostgreSQL** for persistent storage. The schema includes tables for:

- Users (with hashed passwords and premium status)
- Yaps (short messages posted by users)
- Refresh tokens (for managing session state)

SQL boilerplate code is generated using [`sqlc`](https://github.com/kyleconroy/sqlc ), and migrations are handled using [`goose`](https://github.com/pressly/goose ).

All database interactions are type-safe and follow best practices for performance and security.

---

## Code Structure

Key files and packages:

- `main.go`: Entry point; sets up routing, middleware, and handlers.
- `auth.go`: Handles JWT creation/validation, password hashing, and token parsing.
- `internal/database`: Auto-generated models and queries using `sqlc`.
- `handlers/`: All route handlers implement business logic and return structured JSON responses.

---

## Security Practices

- **Password Hashing**: Uses `bcrypt` to securely store user passwords.
- **JWT Tokens**: Signed with HS256 and short expiration times.
- **Refresh Tokens**: Stored in the database and revoked upon logout.
- **Input Sanitization**: Validates and cleans input before saving to the database.
- **Profanity Filter**: Replaces specific words with `****`.

---

## License

This project is licensed under the MIT License – see the [LICENSE](LICENSE) file for details.

---
