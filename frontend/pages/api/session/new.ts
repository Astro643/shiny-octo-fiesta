/**
 * Session new endpoint
 * POST /api/session/new
 * 
 * Called when starting a new conversation.
 * Edge Function: Fast session initialization
 */

import type { NextApiRequest } from 'next';

interface NewResponse {
  ok: boolean;
}

export const config = {
  runtime: 'edge', // Edge runtime for fast operation
};

export default function handler(
  req: NextApiRequest
) {
  if (req.method !== 'POST') {
    return new Response(JSON.stringify({ detail: 'Method Not Allowed' }), {
      status: 405,
      headers: { 'Allow': 'POST', 'Content-Type': 'application/json' },
    });
  }

  try {
    // In production, this would:
    // 1. Initialize a new session
    // 2. Assign an agent if available
    
    // For now, return a mock success
    const response: NewResponse = {
      ok: true,
    };

    return new Response(JSON.stringify(response), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
  } catch (err) {
    console.error('New session error:', err);
    return new Response(JSON.stringify({ detail: 'Could not start new session.' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
