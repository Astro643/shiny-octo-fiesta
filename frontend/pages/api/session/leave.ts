/**
 * Session leave endpoint
 * POST /api/session/leave
 * 
 * Called when the user closes the console to free up the agent.
 * Edge Function: Fast cleanup operation
 */

import type { NextApiRequest } from 'next';

interface LeaveResponse {
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
    // 1. Remove the guild from the active sessions
    // 2. Free up the agent for others
    
    // For now, return a mock success
    const response: LeaveResponse = {
      ok: true,
    };

    return new Response(JSON.stringify(response), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
  } catch (err) {
    console.error('Leave error:', err);
    return new Response(JSON.stringify({ detail: 'Could not process leave request.' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
