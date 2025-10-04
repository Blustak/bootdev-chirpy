# API Endpoints

## /api/healthz

### GET

Readiness Endpoint

#### Response

Status Code: 200
Content-Type: text/plain; charset=utf-8
#### Response Body:

> OK

## /api/users

### POST

Add user endpoint

#### Request structure

Content-Type: application/json
Content-Body:

> {
>   "email": user-email,
>   "password": user-password
> }

### PUT

Update user information endpoint

#### Request Header

Must include:

- Authorization: Bearer \<user-access-token\>

#### Request Structure

> {
>   "email": user-email,
>   "password": user-password
> }

#### Response (both POST and PUT)

Status Code: 201
Content-Type: application/json

Body:
    [[#User]]

## /api/login

### POST

#### Request Structure

> {
>   "email": user-email,
>   "password": user-password
> }

#### Response

Status code: 200
Content-Type: application/json

#### Response body

[[#User]]

## /api/refresh

### POST

Handles refreshing user tokens

#### Request Header

- Authorization: Bearer \<user-refresh-token\>

#### Response

Status-Code: 200
Content-Type: application/json

#### Response-body
> {
>   "token": \<user-access-token\>
> }

## /api/revoke

Revokes user access tokens (long-term refresh ones)

### POST

#### Request Headers

Authorization: Bearer \<user-refresh-token\>

#### Response

Status-code 204


# Common response structures

## User

> {
>   "id": user-id,
>   "created\_at": timestamp,
>   "updated\_at": timestamp,
>   "email": user-email,
>   "token": user-access-token,
>   "refresh\_token": user-refresh-token,
>   "is\_chirpy\_red": boolean, is user premium?
> }
