# Meteor

Meteor is a developer platform in progress for adding in-app voice and video calls, and live streaming, to other applications. It is designed with African products and mobile networks in mind, including low data use and less reliable connections.

## What it provides

Developers manage a Meteor workspace, create projects, and issue separate test and live API keys. The planned integration flow uses Meteor's API to create call rooms and mint short-lived participant tokens; LiveKit handles the audio and video transport, with a JavaScript SDK for connecting end users.

## Current status

The Go API currently supports tenant authentication, project management, and API key creation, listing, and revocation. The dashboard includes authentication flows and a workspace navigation scaffold.

Room and participant APIs, the JavaScript SDK, and usage and billing features are planned and are not yet available.

## Stack

- **API:** Go
- **Data and shared services:** PostgreSQL with row-level security, and Redis
- **Dashboard:** Next.js, TypeScript, and Tailwind CSS
- **Media server:** LiveKit, planned
