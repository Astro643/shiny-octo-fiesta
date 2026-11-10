/**
 * Session ping endpoint
 * POST /api/session/ping
 * 
 * Checks if the session is still active and returns the current state.
 * Edge Function: Fast session validation
 */

import type { NextApiRequest } from 'next';

interface SessionPingResponse {
  state: string;
  agent?: number;
  position?: number;
  ahead?: number;
  waiting?: number;
}

export const config = {
  runtime: 'edge', // Edge runtime for fast session check
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

  // In production, this would check the session cookie
  // For now, return a mock active response
  const response: SessionPingResponse = {
    state: 'active',
    agent: 1,
  };

  return new Response(JSON.stringify(response), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}
