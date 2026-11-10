/**
 * Controller Backend - TypeScript Version
 * Entry point for the Express server
 */

import express, { Express, Request, Response, NextFunction } from 'express';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import morgan from 'morgan';
import { config } from './config';
import { initRedis, getRedis } from './services/redis';
import { HttpError, newErr } from './utils';
import { PreviewRequest, PreviewPlan, SessionPingResponse } from './types';

// Initialize Redis
initRedis().catch(err => {
  console.error('Failed to initialize Redis:', err);
  process.exit(1);
});

// Create Express app
const app: Express = express();
const router = express.Router();

// Middleware
app.use(helmet());
app.use(cors({
  origin: '*', // In production, restrict to your frontend origin
  methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'Authorization'],
}));
app.use(compression());
app.use(express.json({ limit: '10kb' }));
app.use(express.urlencoded({ extended: true }));

// Request logging
app.use(morgan('combined', {
  skip: (req, res) => res.statusCode < 400,
}));

// Health check
app.get('/healthz', (req: Request, res: Response) => {
  res.json({ ok: true });
});

// Mount router
app.use('/', router);

// Error handling middleware
app.use((err: Error, req: Request, res: Response, next: NextFunction) => {
  if (err instanceof HttpError) {
    res.status(err.status).json({ detail: err.detail });
    return;
  }
  
  console.error('Unhandled error:', err);
  res.status(500).json({ detail: 'Internal server error' });
});

// 404 handler
app.use((req: Request, res: Response) => {
  res.status(404).json({ detail: 'Not found' });
});

// Session middleware (placeholder - to be implemented)
function requireGuild(req: Request, res: Response, next: NextFunction) {
  // In a real implementation, this would verify the session cookie
  // and set req.guild_id
  const guildId = req.headers['x-guild-id'] as string || '';
  if (!guildId) {
    res.status(401).json({ detail: 'Session expired. Open a fresh dashboard link.' });
    return;
  }
  (req as any).guild_id = guildId;
  next();
}

// Session routes
router.post('/api/session/ping', async (req: Request, res: Response) => {
  try {
    const guildId = (req as any).guild_id as string;
    if (!guildId) {
      res.status(401).json({ detail: 'Session expired. Open a fresh dashboard link.' });
      return;
    }

    // In a real implementation, this would check the session state
    // For now, return a mock response
    const response: SessionPingResponse = {
      state: 'active',
      agent: 1,
    };
    
    res.json(response);
  } catch (err) {
    res.status(500).json({ detail: 'Service unavailable' });
  }
});

router.post('/api/session/leave', (req: Request, res: Response) => {
  // Session cleanup
  res.json({ ok: true });
});

router.post('/api/session/new', (req: Request, res: Response) => {
  // Start new conversation
  res.json({ ok: true });
});

// Preview route (Task 3)
router.post('/api/preview', requireGuild, async (req: Request, res: Response) => {
  try {
    const guildId = (req as any).guild_id as string;
    const body: PreviewRequest = req.body;
    
    if (!body.prompt || body.prompt.trim() === '') {
      throw newErr(422, 'A prompt is required.');
    }
    
    if (body.prompt.length > config.maxPromptLength) {
      throw newErr(422, `The prompt must be between 1 and ${config.maxPromptLength} characters.`);
    }

    // In a real implementation, this would:
    // 1. Get the agent for this guild
    // 2. Get the server snapshot from Redis
    // 3. Call the AI to generate a preview plan
    // 4. Return the preview
    
    // For now, return a mock preview
    const mockPreview: PreviewPlan = {
      complete: true,
      total_turns: 3,
      steps: [
        {
          turn: 1,
          action: 'create_category_and_channels',
          label: 'Create category Welcome with channels: rules, info',
          description: 'Creating a new category named \'Welcome\' with text channels',
          parameters: {
            category_name: 'Welcome',
            channel_names: ['rules', 'info'],
            channel_type: 'text',
          },
        },
        {
          turn: 2,
          action: 'send_message',
          label: 'Send welcome message to #rules',
          description: 'Sending a welcome message to the rules channel',
          parameters: {
            channel_name: 'rules',
            content: 'Welcome to the server! Please read the rules.',
          },
        },
        {
          turn: 3,
          action: 'send_message',
          label: 'Send info message to #info',
          description: 'Sending an info message to the info channel',
          parameters: {
            channel_name: 'info',
            content: 'Server information and resources.',
          },
        },
      ],
      summary: 'Will create a Welcome category with rules and info channels, then send welcome messages.',
      incomplete: false,
    };
    
    res.json(mockPreview);
  } catch (err) {
    if (err instanceof HttpError) {
      res.status(err.status).json({ detail: err.detail });
      return;
    }
    res.status(500).json({ detail: 'Could not generate preview.' });
  }
});

// Execute route
router.post('/api/execute', requireGuild, async (req: Request, res: Response) => {
  try {
    const guildId = (req as any).guild_id as string;
    const body: { prompt: string } = req.body;
    
    if (!body.prompt || body.prompt.trim() === '') {
      throw newErr(422, 'A prompt is required.');
    }
    
    if (body.prompt.length > config.maxPromptLength) {
      throw newErr(422, `The prompt must be between 1 and ${config.maxPromptLength} characters.`);
    }

    // In a real implementation, this would:
    // 1. Generate a command ID
    // 2. Store the job in Redis
    // 3. Start the Inngest workflow
    // 4. Return the command ID
    
    // For now, return a mock response
    const mockResponse = {
      command_id: `cmd_${Date.now()}`,
      status: 'processing',
      agent: 1,
    };
    
    res.json(mockResponse);
  } catch (err) {
    if (err instanceof HttpError) {
      res.status(err.status).json({ detail: err.detail });
      return;
    }
    res.status(500).json({ detail: 'The task engine is unavailable. Please try again.' });
  }
});

// Status route
router.get('/api/status/:command_id', requireGuild, async (req: Request, res: Response) => {
  try {
    const commandId = req.params.command_id;
    const guildId = (req as any).guild_id as string;
    
    // In a real implementation, this would:
    // 1. Get the status from Redis
    // 2. Return the status, error, and result
    
    // For now, return a mock response
    const mockResponse = {
      command_id: commandId,
      status: 'completed',
      error: null,
      result: 'All steps completed successfully.',
    };
    
    res.json(mockResponse);
  } catch (err) {
    res.status(500).json({ detail: 'Status unavailable.' });
  }
});

// Cancel route
router.post('/api/cancel/:command_id', requireGuild, async (req: Request, res: Response) => {
  try {
    const commandId = req.params.command_id;
    const guildId = (req as any).guild_id as string;
    
    // In a real implementation, this would:
    // 1. Set the cancel flag in Redis
    // 2. Update the status
    
    // For now, return a mock response
    const mockResponse = {
      command_id: commandId,
      status: 'cancelled',
    };
    
    res.json(mockResponse);
  } catch (err) {
    res.status(500).json({ detail: 'Could not cancel right now. Please try again.' });
  }
});

// Public status route
router.get('/api/public/status', async (req: Request, res: Response) => {
  try {
    // In a real implementation, this would:
    // 1. Check the status of all components
    // 2. Return a comprehensive status document
    
    // For now, return a mock response
    const mockResponse = {
      overall: 'operational',
      label: 'All systems operational',
      components: [
        { id: 'website', name: 'Website', status: 'operational', detail: 'Responding' },
        { id: 'database', name: 'Database', status: 'operational', detail: '123 ms' },
        { id: 'bot', name: 'Discord bot', status: 'operational', detail: 'Active · 45 ms' },
        { id: 'engine', name: 'Task engine', status: 'operational', detail: 'Running' },
        { id: 'agent1', name: 'Agent 1', status: 'operational', detail: 'Available · Responding in 0.8s' },
        { id: 'agent2', name: 'Agent 2', status: 'operational', detail: 'Available · Responding in 0.9s' },
        { id: 'agent3', name: 'Agent 3', status: 'operational', detail: 'Available · Responding in 0.7s' },
        { id: 'queue', name: 'Console queue', status: 'operational', detail: '0 of 3 agents in use' },
      ],
      updated: new Date().toISOString(),
    };
    
    res.json(mockResponse);
  } catch (err) {
    res.status(500).json({ detail: 'Status unavailable.' });
  }
});

// Support chat route
router.post('/api/support/chat', async (req: Request, res: Response) => {
  try {
    const body = req.body;
    
    if (!body.message || body.message.trim() === '') {
      throw newErr(422, 'A message is required.');
    }
    
    if (body.message.length > 600) {
      throw newErr(422, 'Message must be 1-600 characters.');
    }

    // In a real implementation, this would:
    // 1. Rate limit the request
    // 2. Get the system status
    // 3. Call the AI to generate a response
    
    // For now, return a mock response
    const mockResponse = {
      reply: 'Thanks for your message! How can I help you with Controller today?',
    };
    
    res.json(mockResponse);
  } catch (err) {
    if (err instanceof HttpError) {
      res.status(err.status).json({ detail: err.detail });
      return;
    }
    res.status(500).json({ detail: 'Support is unavailable right now. Please try again later.' });
  }
});

// Login route
router.get('/login', (req: Request, res: Response) => {
  // In a real implementation, this would:
  // 1. Verify the token from the query string
  // 2. Set the session cookie
  // 3. Redirect to the dashboard
  
  // For now, return a simple message
  res.send('<h1>Login</h1><p>Use /dashboard in Discord to get a magic link.</p>');
});

// Start the server
const PORT = parseInt(config.port) || 8000;

app.listen(PORT, () => {
  console.log(`Controller backend (TypeScript) listening on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/healthz`);
  console.log(`API: http://localhost:${PORT}/api/public/status`);
});

export default app;
