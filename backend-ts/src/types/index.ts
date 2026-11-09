// Preview Types (from Task 3)
export interface PreviewStep {
  turn: number;
  action: string;
  description: string;
  label: string;
  parameters?: StepParameters;
  delay_seconds?: number;
}

export interface PreviewPlan {
  complete: boolean;
  total_turns: number;
  steps: PreviewStep[];
  summary?: string;
  incomplete: boolean;
  question?: string;
}

export interface PreviewRequest {
  prompt: string;
}

// Step Parameters (from planner.go)
export interface StepParameters {
  // Channels / categories
  category_name?: string;
  channel_names?: string[];
  channel_type?: string;
  channel_name?: string;
  channel_id?: number;
  new_name?: string;
  topic?: string;
  slowmode_seconds?: number;
  message_count?: number;
  
  // Roles
  role_name?: string;
  color_hex?: string;
  permissions?: string[];
  
  // Members
  member_id?: string;
  member_name?: string;
  reason?: string;
  delete_message_days?: number;
  timeout_minutes?: number;
  nickname?: string;
  
  // Messages
  title?: string;
  description?: string;
  content?: string;
  
  // General Discord API call
  http_method?: string;
  api_path?: string;
  api_body?: string;
  
  // Non-Discord tools
  query?: string;
  instruction?: string;
  url?: string;
  webhook_method?: string;
  webhook_body?: string;
  email_to?: string;
  email_subject?: string;
  email_body?: string;
}

// Command/Execution Types
export interface CommandRequest {
  prompt: string;
}

export interface ExecuteRequest {
  prompt: string;
}

export interface ExecuteResponse {
  command_id: string;
  status: string;
  agent: number;
}

// Status Types
export interface StatusResponse {
  command_id: string;
  status: string;
  error?: string;
  result?: string;
}

// Session Types
export interface SessionPingResponse {
  state: 'active' | 'waiting' | 'idle_released' | 'full' | 'unavailable';
  agent?: number;
  position?: number;
  ahead?: number;
  waiting?: number;
}

export interface SessionLeaveResponse {
  ok: boolean;
}

// Agent Types
export interface AgentTurn {
  is_safe: boolean;
  rejection_reason?: string;
  done: boolean;
  summary?: string;
  action_name?: string;
  parameters: StepParameters;
  delay_seconds?: number;
}

export interface AgentConfig {
  n: number;
  key: string;
  primary: string;
  fallback: string;
}

// Server Snapshot Types
export interface ServerSnapshot {
  guild_id: string;
  guild_name: string;
  owner_id: string;
  bot_id: string;
  existing_channels: any[];
  existing_roles: any[];
}

// Feed Item Types
export interface FeedItem {
  id: string;
  prompt: string;
  ts: number;
  status: string;
  note: string;
}

// HTTP Error Types
export interface HttpError extends Error {
  status: number;
  detail: string;
}

// Redis Types
export interface RedisConfig {
  url: string;
  token: string;
}

// Database Types
export interface DbConfig {
  url: string;
}

// Configuration Types
export interface AppConfig {
  redis: RedisConfig;
  db?: DbConfig;
  sessionSecret: string;
  cookieSecure: boolean;
  siteName: string;
  supportEmail: string;
  adminAPIKey: string;
  botUrl: string;
  botPrivateKey: string;
  inngestEventKey: string;
  inngestSigningKey: string;
  geminiApiKeys: Record<number, string>;
  primaryModels: Record<number, string>;
  fallbackModels: Record<number, string>;
  webhookAllowedHosts: string[];
  emailRecipients: string[];
  resendApiKey: string;
  emailFrom: string;
}

// Status Component Types
export interface Component {
  id: string;
  name: string;
  status: string;
  detail: string;
}

export interface StatusDoc {
  overall: string;
  label: string;
  components: Component[];
  updated: string;
}

// Support Chat Types
export interface ChatTurn {
  role: 'user' | 'assistant';
  text: string;
}

export interface SupportChatRequest {
  message: string;
  history: ChatTurn[];
}

// Admin Types
export interface AdminCommandRequest {
  command?: string;
}

// Human Support Types
export interface HumanMessage {
  id: string;
  sender: string;
  content: string;
  timestamp: number;
}

// Context Types
export interface ContextEntry {
  p: string; // prompt
  s: string; // status
  r?: string; // result
}
