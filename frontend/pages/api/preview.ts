/**
 * Preview (Dry Run) endpoint
 * POST /api/preview
 * 
 * Generates a preview plan for a given prompt without executing it.
 * This is the "dry run" feature from Task 3.
 * 
 * Serverless Function: Needs AI calls (9-second timeout)
 */

import type { NextApiRequest, NextApiResponse } from 'next';

// Explicitly set as serverless (default, but good to be clear)
// This allows 60-second timeout for AI operations
export const config = {
  runtime: 'nodejs', // Serverless Node.js runtime (default)
  // Note: For AI operations, we need the full Node.js environment
  // Edge runtime doesn't support AI SDKs and has shorter timeout
};

interface PreviewStep {
  turn: number;
  action: string;
  label: string;
  description: string;
  parameters?: Record<string, any>;
  delay_seconds?: number;
}

interface PreviewPlan {
  complete: boolean;
  total_turns: number;
  steps: PreviewStep[];
  summary?: string;
  incomplete: boolean;
  question?: string;
}

interface PreviewRequest {
  prompt: string;
}

// Generate a clear English label for a step
function generateStepLabel(action: string, params: Record<string, any>): string {
  const paramOrDefault = (p: any, defaultVal: string): string => {
    return p !== undefined && p !== null ? String(p) : defaultVal;
  };

  const paramOrDefaultInt = (p: any, defaultVal: number): number => {
    return p !== undefined && p !== null ? Number(p) : defaultVal;
  };

  const truncateString = (s: string, maxLen: number): string => {
    return s.length <= maxLen ? s : s.substring(0, maxLen) + '...';
  };

  switch (action) {
    case 'create_category_and_channels':
      const cat = paramOrDefault(params.category_name, 'category');
      const channels = params.channel_names && Array.isArray(params.channel_names) && params.channel_names.length > 0
        ? ' with channels: ' + params.channel_names.join(', ')
        : '';
      return `Create category ${cat}${channels}`;
    case 'create_custom_role':
      return `Create role ${paramOrDefault(params.role_name, 'role')}`;
    case 'delete_channel':
      return `Delete channel ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))}`;
    case 'delete_role':
      return `Delete role ${paramOrDefault(params.role_name, 'role')}`;
    case 'rename_channel':
      return `Rename channel ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))} to ${paramOrDefault(params.new_name, 'new name')}`;
    case 'rename_role':
      return `Rename role ${paramOrDefault(params.role_name, 'role')} to ${paramOrDefault(params.new_name, 'new name')}`;
    case 'send_message':
      return `Send message to ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))}`;
    case 'send_formatted_embed':
      return `Send embed to ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))}`;
    case 'kick_member':
      return `Kick member ${paramOrDefault(params.member_name, paramOrDefault(params.member_id, 'member'))}`;
    case 'ban_member':
      return `Ban member ${paramOrDefault(params.member_name, paramOrDefault(params.member_id, 'member'))}`;
    case 'unban_member':
      return `Unban member ${paramOrDefault(params.member_id, 'member')}`;
    case 'timeout_member':
      return `Timeout member ${paramOrDefault(params.member_name, paramOrDefault(params.member_id, 'member'))} for ${paramOrDefaultInt(params.timeout_minutes, 0)} minutes`;
    case 'remove_timeout':
      return `Remove timeout from ${paramOrDefault(params.member_name, paramOrDefault(params.member_id, 'member'))}`;
    case 'assign_role':
      return `Assign role ${paramOrDefault(params.role_name, 'role')} to ${paramOrDefault(params.member_name, paramOrDefault(params.member_id, 'member'))}`;
    case 'remove_role':
      return `Remove role ${paramOrDefault(params.role_name, 'role')} from ${paramOrDefault(params.member_name, paramOrDefault(params.member_id, 'member'))}`;
    case 'set_channel_topic':
      return `Set topic for ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))}`;
    case 'set_slowmode':
      return `Set slowmode for ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))}`;
    case 'purge_messages':
      return `Purge ${paramOrDefaultInt(params.message_count, 0)} messages from ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))}`;
    case 'set_nickname':
      return `Set nickname for ${paramOrDefault(params.member_name, paramOrDefault(params.member_id, 'member'))}`;
    case 'web_research':
      return `Search web for: ${paramOrDefault(params.query, 'query')}`;
    case 'write_text':
      return `Write text: ${truncateString(paramOrDefault(params.instruction, 'instruction'), 50)}`;
    case 'call_webhook':
      return `Call webhook at ${paramOrDefault(params.url, 'URL')}`;
    case 'send_email':
      return `Send email to ${paramOrDefault(params.email_to, 'recipient')}`;
    case 'discord_api':
      return `Discord API call: ${paramOrDefault(params.http_method, 'method')} ${paramOrDefault(params.api_path, 'path')}`;
    case 'lock_channel_permissions':
      return `Lock permissions for ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))}`;
    case 'unlock_channel_permissions':
      return `Unlock permissions for ${paramOrDefault(params.channel_name, paramOrDefault(params.channel_id, 'channel'))}`;
    default:
      return action || 'Unknown action';
  }
}

// Generate a detailed description for a step
function generateStepDescription(action: string, params: Record<string, any>): string {
  const paramOrDefault = (p: any, defaultVal: string): string => {
    return p !== undefined && p !== null ? String(p) : defaultVal;
  };

  switch (action) {
    case 'create_category_and_channels':
      const cat = paramOrDefault(params.category_name, 'category');
      const channels = params.channel_names && Array.isArray(params.channel_names) && params.channel_names.length > 0
        ? '. Channels: ' + params.channel_names.join(', ')
        : '';
      const channelType = paramOrDefault(params.channel_type, 'text');
      return `Creating a new category named '${cat}' with ${channelType} channels${channels}`;
    case 'web_research':
      return `Searching the web for: ${paramOrDefault(params.query, 'query')}`;
    case 'write_text':
      return `Writing text based on: ${paramOrDefault(params.instruction, 'instruction').substring(0, 100)}`;
    case 'call_webhook':
      return `Calling webhook at ${paramOrDefault(params.url, 'URL')} with ${paramOrDefault(params.webhook_method, 'POST')} method`;
    case 'send_email':
      return `Sending email to ${paramOrDefault(params.email_to, 'recipient')} with subject: ${paramOrDefault(params.email_subject, 'subject')}`;
    default:
      return generateStepLabel(action, params);
  }
}

// Mock preview generator - in production, this would call the AI
function generateMockPreview(prompt: string): PreviewPlan {
  const lowercasePrompt = prompt.toLowerCase();
  
  // Analyze the prompt to determine what actions are needed
  const steps: PreviewStep[] = [];
  let summary = '';
  let complete = true;
  let question = '';

  if (lowercasePrompt.includes('create') && lowercasePrompt.includes('category')) {
    steps.push({
      turn: 1,
      action: 'create_category_and_channels',
      label: generateStepLabel('create_category_and_channels', {
        category_name: 'New Category',
        channel_names: ['general', 'info'],
        channel_type: 'text',
      }),
      description: generateStepDescription('create_category_and_channels', {
        category_name: 'New Category',
        channel_names: ['general', 'info'],
        channel_type: 'text',
      }),
      parameters: {
        category_name: 'New Category',
        channel_names: ['general', 'info'],
        channel_type: 'text',
      },
    });
    summary = 'Will create a new category with channels.';
  } else if (lowercasePrompt.includes('send') && lowercasePrompt.includes('message')) {
    steps.push({
      turn: 1,
      action: 'send_message',
      label: generateStepLabel('send_message', {
        channel_name: 'general',
        content: 'Hello everyone!',
      }),
      description: generateStepDescription('send_message', {
        channel_name: 'general',
        content: 'Hello everyone!',
      }),
      parameters: {
        channel_name: 'general',
        content: 'Hello everyone!',
      },
    });
    summary = 'Will send a message to the general channel.';
  } else if (lowercasePrompt.includes('ban') || lowercasePrompt.includes('kick')) {
    steps.push({
      turn: 1,
      action: lowercasePrompt.includes('ban') ? 'ban_member' : 'kick_member',
      label: generateStepLabel(lowercasePrompt.includes('ban') ? 'ban_member' : 'kick_member', {
        member_name: 'user123',
        reason: 'Breaking rules',
      }),
      description: generateStepDescription(lowercasePrompt.includes('ban') ? 'ban_member' : 'kick_member', {
        member_name: 'user123',
        reason: 'Breaking rules',
      }),
      parameters: {
        member_name: 'user123',
        reason: 'Breaking rules',
      },
    });
    summary = `Will ${lowercasePrompt.includes('ban') ? 'ban' : 'kick'} the specified member.`;
  } else if (lowercasePrompt.includes('role')) {
    steps.push({
      turn: 1,
      action: 'create_custom_role',
      label: generateStepLabel('create_custom_role', {
        role_name: 'New Role',
        color_hex: '#5865F2',
      }),
      description: generateStepDescription('create_custom_role', {
        role_name: 'New Role',
        color_hex: '#5865F2',
      }),
      parameters: {
        role_name: 'New Role',
        color_hex: '#5865F2',
      },
    });
    summary = 'Will create a new custom role.';
  } else if (lowercasePrompt.includes('web') || lowercasePrompt.includes('search') || lowercasePrompt.includes('research')) {
    steps.push({
      turn: 1,
      action: 'web_research',
      label: generateStepLabel('web_research', {
        query: prompt,
      }),
      description: generateStepDescription('web_research', {
        query: prompt,
      }),
      parameters: {
        query: prompt,
      },
    });
    summary = 'Will search the web for information.';
  } else {
    // Generic fallback - ask for clarification
    complete = false;
    question = 'Can you clarify your request? I need more details to plan all the steps.';
    summary = 'I need more information to understand what you want to do.';
    
    // Try to extract some intent
    if (prompt.length > 10) {
      steps.push({
        turn: 1,
        action: 'unknown',
        label: 'Understanding request...',
        description: `Analyzing your request: ${prompt.substring(0, 100)}`,
        parameters: {},
      });
    }
  }

  return {
    complete,
    total_turns: steps.length,
    steps,
    summary,
    incomplete: !complete,
    question: complete ? undefined : question,
  };
}

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<PreviewPlan | { detail: string }>
) {
  if (req.method !== 'POST') {
    res.setHeader('Allow', ['POST']);
    return res.status(405).json({ detail: 'Method Not Allowed' });
  }

  try {
    const body: PreviewRequest = req.body;
    
    if (!body.prompt || body.prompt.trim() === '') {
      return res.status(422).json({ detail: 'A prompt is required.' } as any);
    }
    
    if (body.prompt.length > 2000) {
      return res.status(422).json({ detail: 'The prompt must be between 1 and 2000 characters.' } as any);
    }

    // Generate preview - in production, this would call the AI service
    // For now, use the mock generator
    const preview: PreviewPlan = generateMockPreview(body.prompt);
    
    return res.status(200).json(preview);
  } catch (err) {
    console.error('Preview error:', err);
    return res.status(500).json({ detail: 'Could not generate preview.' } as any);
  }
}
