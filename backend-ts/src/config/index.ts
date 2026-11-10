import dotenv from 'dotenv';

dotenv.config();

// Default values
const DEFAULT_PRIMARY_MODEL = 'gemini-3.6-flash';
const DEFAULT_FALLBACK_MODEL = 'gemini-3.5-flash-lite';
const DEFAULT_SITE_NAME = 'Controller';
const DEFAULT_PORT = '8000';
const DEFAULT_COOKIE_SECURE = 'true';

// Helper function to get environment variable with default
function getEnv(name: string, defaultValue: string = ''): string {
  return process.env[name]?.trim() || defaultValue;
}

// Helper function to get boolean from environment
function getEnvBool(name: string, defaultValue: boolean = true): boolean {
  const value = process.env[name]?.trim().toLowerCase();
  if (value === 'false' || value === '0') return false;
  if (value === 'true' || value === '1') return true;
  return defaultValue;
}

// Helper function to get array from comma-separated environment variable
function getEnvArray(name: string): string[] {
  const value = process.env[name]?.trim() || '';
  return value.split(',').map(item => item.trim()).filter(item => item !== '');
}

// Agent Configuration
export interface AgentConfig {
  n: number;
  key: string;
  primary: string;
  fallback: string;
}

// Main configuration
export const config = {
  // Server configuration
  port: getEnv('PORT', DEFAULT_PORT),
  
  // Redis configuration
  redis: {
    url: getEnv('REDIS_URL', getEnv('UPSTASH_REDIS_REST_URL')),
    token: getEnv('REDIS_TOKEN', getEnv('UPSTASH_REDIS_REST_TOKEN')),
  },
  
  // Session configuration
  sessionSecret: getEnv('SESSION_SECRET'),
  cookieSecure: getEnvBool('COOKIE_SECURE', DEFAULT_COOKIE_SECURE === 'true'),
  
  // Site configuration
  siteName: getEnv('SITE_NAME', DEFAULT_SITE_NAME),
  supportEmail: getEnv('SUPPORT_EMAIL'),
  adminAPIKey: getEnv('ADMIN_API_KEY'),
  
  // Bot configuration
  botUrl: getEnv('BOT_URL'),
  botPrivateKey: getEnv('BOT_PRIVATE_KEY'),
  
  // Inngest configuration
  inngestEventKey: getEnv('INNGEST_EVENT_KEY'),
  inngestSigningKey: getEnv('INNGEST_SIGNING_KEY'),
  inngestDev: getEnvBool('INNGEST_DEV', false),
  
  // Agent configuration (1-4)
  agents: {
    1: {
      key: getEnv('GEMINI_API_KEY_1'),
      primary: getEnv('PRIMARY_MODEL_1', getEnv('PRIMARY_MODEL', DEFAULT_PRIMARY_MODEL)),
      fallback: getEnv('FALLBACK_MODEL_1', getEnv('FALLBACK_MODEL', DEFAULT_FALLBACK_MODEL)),
    },
    2: {
      key: getEnv('GEMINI_API_KEY_2'),
      primary: getEnv('PRIMARY_MODEL_2', getEnv('PRIMARY_MODEL', DEFAULT_PRIMARY_MODEL)),
      fallback: getEnv('FALLBACK_MODEL_2', getEnv('FALLBACK_MODEL', DEFAULT_FALLBACK_MODEL)),
    },
    3: {
      key: getEnv('GEMINI_API_KEY_3'),
      primary: getEnv('PRIMARY_MODEL_3', getEnv('PRIMARY_MODEL', DEFAULT_PRIMARY_MODEL)),
      fallback: getEnv('FALLBACK_MODEL_3', getEnv('FALLBACK_MODEL', DEFAULT_FALLBACK_MODEL)),
    },
    4: {
      key: getEnv('GEMINI_API_KEY_4'),
      primary: getEnv('PRIMARY_MODEL_4', getEnv('PRIMARY_MODEL', DEFAULT_PRIMARY_MODEL)),
      fallback: getEnv('FALLBACK_MODEL_4', getEnv('FALLBACK_MODEL', DEFAULT_FALLBACK_MODEL)),
    },
  },
  
  // Tool configuration
  webhookAllowedHosts: getEnvArray('WEBHOOK_ALLOWED_HOSTS'),
  emailRecipients: getEnvArray('EMAIL_ALLOWED_TO'),
  resendApiKey: getEnv('RESEND_API_KEY'),
  emailFrom: getEnv('EMAIL_FROM'),
  
  // Database configuration (optional)
  dbUrl: getEnv('DATABASE_URL'),
  historyDays: parseInt(getEnv('HISTORY_DAYS', '30')),
  
  // Timeout configuration (Task 3: 9-second limit)
  microTaskTimeout: 9000, // 9 seconds in milliseconds
  
  // Workflow configuration
  maxTurns: 10,
  maxDiscordSteps: 25,
  maxDelayStep: 6 * 3600,
  maxDelayTotal: 12 * 3600,
  maxPromptLength: 2000,
  
  // Session configuration
  maxPendingCommands: 10,
  leaseSeconds: 120,
  leaveGraceSeconds: 15,
  waitAliveSeconds: 20,
  idleSeconds: 20 * 60,
  maxWaiting: 50,
  
  // Status probe
  statusProbeSeconds: parseInt(getEnv('STATUS_PROBE_SECONDS', '900')),
};

// Session agents (1, 2, 3)
export const sessionAgents = [1, 2, 3];

// Retry delays for agent calls
export const retryDelays = [1.5, 4.0]; // seconds

// Validate configuration
function validateConfig() {
  if (!config.redis.url || !config.redis.token) {
    console.warn('Warning: REDIS_URL and REDIS_TOKEN are required for Redis operations');
  }
  
  if (!config.botPrivateKey) {
    console.warn('Warning: BOT_PRIVATE_KEY is required for bot communication');
  }
  
  if (!config.inngestEventKey || !config.inngestSigningKey) {
    console.warn('Warning: INNGEST_EVENT_KEY and INNGEST_SIGNING_KEY are required for task engine');
  }
  
  // Check at least one agent is configured
  const configuredAgents = Object.values(config.agents).filter(a => a.key !== '');
  if (configuredAgents.length === 0) {
    console.warn('Warning: At least one GEMINI_API_KEY is required');
  }
}

// Initialize configuration validation
validateConfig();

export default config;
