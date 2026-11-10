# Controller Backend - TypeScript

TypeScript rewrite of the Controller backend (originally in Go).

## Status

**IN PROGRESS** - Core infrastructure is in place, but business logic needs to be ported from the Go version.

## What's Done

### Core Infrastructure
- ✅ Express server with middleware (helmet, cors, compression, morgan)
- ✅ Configuration management from environment variables
- ✅ TypeScript type definitions for all major types
- ✅ Utility functions (text, error, time helpers)
- ✅ Redis service (placeholder - needs Upstash REST implementation)

### API Endpoints (Mock implementations)
- ✅ `/healthz` - Health check
- ✅ `/api/session/ping` - Session status
- ✅ `/api/session/leave` - End session
- ✅ `/api/session/new` - New conversation
- ✅ `/api/preview` - Dry run preview (Task 3)
- ✅ `/api/execute` - Execute command
- ✅ `/api/status/:command_id` - Command status
- ✅ `/api/cancel/:command_id` - Cancel command
- ✅ `/api/public/status` - Public system status
- ✅ `/api/support/chat` - Support chat
- ✅ `/login` - Login page

### Task 3 Features Implemented
- ✅ Generic status messages (`completed`, `rejected`)
- ✅ 9-second timeout constant
- ✅ Preview endpoint with clear English labels
- ✅ Mock preview responses for testing

## What's Missing

### Services to Create
- [ ] Agent service (Gemini API calls)
- [ ] Workflow service (Inngest integration)
- [ ] Session service (session management)
- [ ] Context service (conversation history)
- [ ] Status service (system monitoring)
- [ ] DB service (PostgreSQL)
- [ ] Crypto service (Ed25519 signing)
- [ ] Tools service (web research, email, webhooks)

### Middleware to Create
- [ ] Authentication middleware
- [ ] Error handling middleware (partially done)
- [ ] Rate limiting middleware

### Routes to Complete
- [ ] Admin routes
- [ ] Human support routes
- [ ] Operator routes

## Project Structure

```
backend-ts/
├── src/
│   ├── index.ts              # Express server entry point
│   ├── config/
│   │   └── index.ts          # Configuration management
│   ├── services/
│   │   └── redis.ts          # Redis client
│   ├── routes/
│   │   └── (to be created)   # Route handlers
│   ├── middleware/
│   │   └── (to be created)   # Express middleware
│   ├── types/
│   │   └── index.ts          # TypeScript interfaces
│   └── utils/
│       └── index.ts          # Utility functions
├── package.json
├── tsconfig.json
└── README.md
```

## Setup

### Prerequisites
- Node.js 18+
- npm or yarn

### Install Dependencies
```bash
cd backend-ts
npm install
```

### Configure Environment
Create a `.env` file with the following variables:
```bash
# Server
PORT=8000

# Redis (Upstash)
REDIS_URL=https://your-upstash-redis-url
REDIS_TOKEN=your-upstash-token

# Session
SESSION_SECRET=your-secret-key
COOKIE_SECURE=true

# Site
SITE_NAME=Controller
SUPPORT_EMAIL=support@example.com
ADMIN_API_KEY=your-admin-key

# Bot
BOT_URL=https://your-bot-url
BOT_PRIVATE_KEY=your-ed25519-private-key

# Inngest
INNGEST_EVENT_KEY=your-event-key
INNGEST_SIGNING_KEY=your-signing-key
INNGEST_DEV=false

# AI Agents (1-4)
GEMINI_API_KEY_1=your-gemini-key-1
PRIMARY_MODEL_1=gemini-3.6-flash
FALLBACK_MODEL_1=gemini-3.5-flash-lite

GEMINI_API_KEY_2=your-gemini-key-2
PRIMARY_MODEL_2=gemini-3.6-flash
FALLBACK_MODEL_2=gemini-3.5-flash-lite

# Tools (optional)
WEBHOOK_ALLOWED_HOSTS=api.example.com,hooks.example.com
EMAIL_ALLOWED_TO=user@example.com
RESEND_API_KEY=your-resend-key
EMAIL_FROM=noreply@example.com

# Database (optional)
DATABASE_URL=postgresql://user:password@host/database
HISTORY_DAYS=30
```

### Run the Server
```bash
# Development
npm run dev

# Production
npm run build
npm start
```

## Next Steps

1. **Implement Agent Service** - Port the Gemini API calls from `agents.go`
2. **Implement Workflow Service** - Port the Inngest integration from `workflow.go`
3. **Complete Route Handlers** - Fill in the actual logic for all endpoints
4. **Add Authentication** - Implement session cookie verification
5. **Add Rate Limiting** - Prevent abuse
6. **Add Error Handling** - Improve error responses
7. **Test Everything** - Ensure all features work correctly

## Porting Notes

### From Go to TypeScript

1. **Gin → Express**: Replace Gin router with Express
2. **Redis**: Use `redis` or `@upstash/redis` package
3. **PostgreSQL**: Use `pg` package
4. **HTTP Client**: Use `axios` or native `fetch`
5. **JSON**: Use native `JSON` methods
6. **Error Handling**: Use try/catch instead of Go's error returns
7. **Concurrency**: Use Promises instead of goroutines

### Key Differences

- **Async/Await**: TypeScript uses async/await for asynchronous operations
- **Type System**: TypeScript has a strong type system (use it!)
- **Middleware**: Express middleware is different from Gin middleware
- **Error Handling**: Express uses error-handling middleware

## License

MIT License
