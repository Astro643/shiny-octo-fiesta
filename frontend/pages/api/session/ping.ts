/**
 * Session ping endpoint
 * POST /api/session/ping
 * 
 * Checks if the session is still active and returns the current state.
 * Standard Node.js Vercel Function (Fluid Compute)
 */

import type { NextApiRequest, NextApiResponse } from 'next';

interface SessionPingResponse {
  state: string;
  agent?: number;
  position?: number;
  ahead?: number;
  waiting?: number;
}

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<SessionPingResponse | { detail: string }>
) {
  if (req.method !== 'POST') {
    res.setHeader('Allow', ['POST']);
    return res.status(405).json({ detail: 'Method Not Allowed' });
  }

  try {
    // In production, this would check the session cookie
    // For now, return a mock active response
    const response: SessionPingResponse = {
      state: 'active',
      agent: 1,
    };

    return res.status(200).json(response);
  } catch (err) {
    console.error('Ping error:', err);
    return res.status(500).json({ detail: 'Service unavailable' });
  }
}
