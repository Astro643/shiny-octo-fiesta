# Vercel Deployment Guide

This guide explains how to deploy the Controller application to Vercel.

## Overview

The Controller application is now optimized for Vercel deployment with:
- Next.js 14 frontend
- API routes for backend functionality
- Serverless functions for all backend endpoints

## Quick Deploy

### Option 1: Deploy via Vercel CLI

```bash
# Install Vercel CLI
npm install -g vercel

# Navigate to the frontend directory
cd frontend

# Deploy
vercel
```

Follow the prompts to link your Vercel account and deploy.

### Option 2: Deploy via Vercel Dashboard

1. Go to [https://vercel.com](https://vercel.com)
2. Click "Add New" -> "Project"
3. Import the `shiny-octo-fiesta` repository
4. Configure the project:
   - **Framework Preset**: Next.js
   - **Root Directory**: `frontend`
   - **Build Command**: `npm run build`
   - **Output Directory**: `.next`
   - **Install Command**: `npm install`
5. Click "Deploy"

## Project Structure for Vercel

```
shiny-octo-fiesta/
├── frontend/                    # Next.js application (deployed to Vercel)
│   ├── pages/                   # Next.js pages
│   │   ├── api/                # API routes (serverless functions)
│   │   │   ├── preview.ts      # Preview/dry run endpoint
│   │   │   ├── execute.ts      # Execute command endpoint
│   │   │   ├── status/         # Command status endpoint
│   │   │   │   └── [command_id].ts
│   │   │   ├── cancel/         # Cancel command endpoint
│   │   │   │   └── [command_id].ts
│   │   │   ├── session/        # Session management
│   │   │   │   ├── ping.ts
│   │   │   │   ├── leave.ts
│   │   │   │   └── new.ts
│   │   │   └── login.ts        # Authentication endpoint
│   │   ├── dashboard.tsx       # Main console page
│   │   ├── index.tsx          # Landing page
│   │   └── ...                # Other pages
│   ├── package.json
│   ├── next.config.js
│   └── tsconfig.json
├── vercel.json                  # Vercel configuration
└── README.md
```

## Configuration

### Environment Variables

Create a `.env.local` file in the `frontend` directory:

```bash
# Next.js configuration
NEXT_PUBLIC_APP_URL=https://your-app.vercel.app
NEXT_PUBLIC_API_URL=https://your-app.vercel.app/api

# For development
NODE_ENV=development
```

In the Vercel dashboard, add these environment variables:
- `NODE_ENV`: `production`
- `NEXT_PUBLIC_APP_URL`: Your Vercel deployment URL

### Vercel Configuration (vercel.json)

The `vercel.json` file is already configured with:
- Next.js framework preset
- Proper routing for API endpoints
- Multiple regions for global deployment

## API Routes

All API routes are implemented as Next.js API routes (serverless functions):

### Authentication
- `POST /api/login` - Authenticate and set session cookie

### Session Management
- `POST /api/session/ping` - Check session status
- `POST /api/session/leave` - Free agent when closing console
- `POST /api/session/new` - Start new conversation

### Commands
- `POST /api/preview` - Generate preview plan (dry run)
- `POST /api/execute` - Queue command for execution
- `GET /api/status/:command_id` - Check command status
- `POST /api/cancel/:command_id` - Cancel waiting command

### Health
- `GET /api/healthz` - Health check endpoint

## Task 3 Features Implemented

### 1. Generic Status Messages
All status messages are now generic and not Discord-specific:
- `completed` instead of `executed`
- `rejected` for audit failures
- Clear, action-agnostic labels

### 2. Dry Run Preview
- `POST /api/preview` endpoint generates a complete plan
- Returns structured `PreviewPlan` with steps, labels, and descriptions
- Each step has a clear English label (e.g., "Create category Welcome with channels: rules, info")
- Supports incomplete previews with questions

### 3. 9-Second Strict Limit
- Mock implementations simulate 9-second timeout
- In production, integrate with actual AI services with 9-second timeout
- Each micro-task (step) is limited to 9 seconds

### 4. Preview Auto-Execute
- Frontend shows preview modal for 9 seconds
- Auto-executes if preview is complete
- Asks question if preview is incomplete

## Development

### Run Locally

```bash
# Install dependencies
cd frontend
npm install

# Start development server
npm run dev
```

The app will be available at `http://localhost:3000`

### Test API Routes

```bash
# Preview endpoint
curl -X POST http://localhost:3000/api/preview \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Create a welcome channel"}'

# Execute endpoint
curl -X POST http://localhost:3000/api/execute \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Create a welcome channel"}'

# Status endpoint
curl http://localhost:3000/api/status/COMMAND_ID
```

## Production Deployment

### Custom Domains

To add a custom domain:
1. Go to your project in Vercel dashboard
2. Click "Settings" -> "Domains"
3. Add your custom domain
4. Follow the DNS configuration instructions

### SSL Certificates

Vercel automatically provisions SSL certificates for all domains.

### Monitoring

Vercel provides built-in monitoring:
- Real-time logs
- Performance metrics
- Error tracking
- Usage analytics

## Performance Optimization

### Next.js Optimizations
- Automatic static optimization
- Image optimization (if using Next.js Image component)
- Code splitting
- Tree shaking
- Compression

### Vercel-Specific Optimizations
- Edge Network for faster global delivery
- Automatic caching
- Serverless functions for API routes
- Instant deployment with preview URLs

## Troubleshooting

### Common Issues

1. **API routes not working**
   - Ensure you're calling the correct URL
   - Check the Vercel deployment logs
   - Verify environment variables are set

2. **Session cookie not persisting**
   - Ensure `NODE_ENV` is set to `production` in Vercel
   - Check cookie settings (secure, sameSite, etc.)

3. **Build failures**
   - Run `npm install` locally to verify dependencies
   - Check for TypeScript errors with `npx tsc --noEmit`

4. **Preview not showing**
   - Verify the `/api/preview` endpoint is returning valid JSON
   - Check browser console for errors

### Debug Mode

Add `DEBUG=controller:*` environment variable to enable debug logging.

## Scaling

Vercel automatically scales:
- Serverless functions scale to zero when idle
- No cold start issues with Vercel's edge network
- Automatic load balancing
- Global distribution

## Cost

Vercel's free tier includes:
- Unlimited deployments
- 100GB bandwidth/month
- Serverless functions with generous limits
- Automatic HTTPS

For production use, consider the Pro or Enterprise plan for:
- Higher limits
- Faster builds
- Advanced monitoring
- Priority support

## Migration from Go Backend

If you were previously using the Go backend (`backend/` directory):

1. The TypeScript backend (`backend-ts/`) contains equivalent logic
2. API routes in `frontend/pages/api/` implement the same endpoints
3. For production, you can:
   - Use the Vercel API routes (recommended for Vercel deployment)
   - Deploy the Go backend separately and update API URLs
   - Port the Go backend logic to the TypeScript backend

## Next Steps

1. **Test thoroughly** - Verify all features work in development
2. **Deploy to Vercel** - Use the Vercel CLI or dashboard
3. **Configure environment variables** - Set up production environment
4. **Monitor** - Check Vercel dashboard for errors and performance
5. **Iterate** - Deploy updates with `vercel --prod`

## Support

For issues with Vercel deployment:
- Check [Vercel Documentation](https://vercel.com/docs)
- Visit [Vercel Community](https://github.com/vercel/community)
- Contact [Vercel Support](https://vercel.com/support)

For issues with the Controller application:
- Check the main [README.md](README.md)
- Review the [Task Details](README.md#task-details)
