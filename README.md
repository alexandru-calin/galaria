# galaria

This is a Web Application built with Go that allows people to create and share image galleries

- [Features](#features)
- [Getting Started](#getting-started)

## Features

- MVC architectural pattern
- Uploading images & organizing
- Session based authentication system (1 session per user)
- CSRF protection
- Server-side rendering

## Getting Started

### 1. Clone the Repository
```
git clone https://github.com/alexandru-calin/galaria`
cd galaria
```

### 2. Set up environment variables
Create an .env file and add the environment variables to it.

There is an .env.example file included if you don't know what variables need to be set.

### 3. Build the docker container
```
docker compose -f compose.yaml -f compose.production.yaml up --build
```

### 4. Use the application
Open your browser and navigate to http://localhost

<br>
You should see the home page of the web application. From here, users can sign up, log in, and interact with the platform.
