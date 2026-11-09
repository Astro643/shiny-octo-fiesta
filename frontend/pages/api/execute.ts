/**
 * Execute endpoint
 * POST /api/execute
 * 
 * Queues a command for execution after preview approval.
 */

import type { NextApiRequest, NextApiResponse } from 'next';
import { v4 as uuidv4 } from 'uuid';

interface ExecuteRequest {
  prompt: string;
}

interface ExecuteResponse {
  command_id: string;
  status: string;
}

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<ExecuteResponse | { detail: string }>
) {
  if (req.method !== 'POST') {
    res.setHeader('Allow', ['POST']);
    return res.status(405).json({ detail: 'Method Not Allowed' });
  }

  try {
    const body: ExecuteRequest = req.body;
    
    if (!body.prompt || body.prompt.trim() === '') {
      return res.status(422).json({ detail: 'A prompt is required.' } as any);
    }
    
    if (body.prompt.length > 2000) {
      return res.status(422).json({ detail: 'The prompt must be between 1 and 2000 characters.' } as any);
    }

    // Generate a unique command ID
    const commandId = uuidv4();
    
    // In production, this would:
    // 1. Store the job in Redis with the guild ID and prompt
    // 2. Start the Inngest workflow
    // 3. Return the command ID for status tracking
    
    // For now, return a mock response
    const response: ExecuteResponse = {
      command_id: commandId,
      status: 'processing',
    };

    return res.status(200).json(response);
  } catch (err) {
    console.error('Execute error:', err);
    return res.status(500).json({ detail: 'Could not queue that goal.' } as any);
  }
}
