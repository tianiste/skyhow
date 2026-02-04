# skyhow

This is a hobby project to build a single place for Skyblock guides and related information that is currently spread across Discord servers and random sites.

Right now the project is focused on getting the backend architecture right before worrying about frontend or features.



## what the project currently does

### authentication  
Users can log in using Discord OAuth.  
When a user logs in, the backend creates or updates a user record in the database and creates a server-side session.  
The session id is stored in a secure cookie `sb_session`.  

Logging out deletes the session from the database and clears the cookie.

All authentication is handled by the backend. There is no third-party auth service involved.


### sessions  
Sessions are stored in the database.  
Each session links a random session id to a user and an expiration time.

On every request, the backend checks whether the session cookie exists and whether it is valid.  
If the session is missing, expired, or invalid, the cookie is cleared automatically.


### middleware  
AuthMiddleware runs on every request.

It:
- reads the session cookie
- validates the session in the database
- loads the user
- attaches the user to the request context

If anything is invalid, the request continues as unauthenticated.

RequireAuth is a small middleware used on protected routes.  
It simply blocks the request if no authenticated user is present.


### users  
Users are stored internally with UUIDs.  
User data comes from OAuth providers (Discord for now).

Each user has:
- a display name
- an optional avatar url
- a role (user / editor / admin)
- an active flag


### guides  
Guides are stored in the database.

A guide has:
- a creator (user)
- a title
- markdown content
- a status (draft or published)
- tags
- timestamps

Rules:
- new guides always start as draft
- draft guides are private
- published guides are public
- only the creator can edit or delete their guide (for now)


### tags  
Tags are stored in their own table and linked to guides through a join table.

Tags are:
- lowercase
- unique
- reusable across guides

They are used for searching and filtering guides.


### api endpoints

public:
- GET /healthz
- GET /me
- GET /api/guides
- GET /api/guides/:id

auth:
- GET /auth/discord/start
- GET /auth/discord/callback
- POST /auth/logout

protected:
- POST /api/guides
- PUT /api/guides/:id
- POST /api/guides/:id/publish
- POST /api/guides/:id/unpublish
- DELETE /api/guides/:id



