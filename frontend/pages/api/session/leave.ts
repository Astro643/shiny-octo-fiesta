/**
 * Session leave endpoint
 * POST /api/session/leave
 * 
 * Called when the user closes the console to free up the agent.
 * Standard Node.js Vercel Function (Fluid Compute)
 */

import type { NextApiRequest, NextApiResponse } from 'next';

interface LeaveResponse {
  ok: boolean;
}

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<LeaveResponse | { detail: string }>
) {
  if (req.method !== 'POST') {
    res.setHeader('Allow', ['POST']);
    return res.status(405).json({ detail: 'Method Not Allowed' });
  }

  try {
    // In production, this would:
    // 1. Remove the guild from the active sessions
    // 2. Free up the agent for others
    
    // For now, return a mock success
    const response: LeaveResponse = {
      ok: true,
    };

    return res.status(200).json(response);
  } catch (err) {
    console.error('Leave error:', err);
    return res.status(500).json({ detail: 'Could not process leave request.' });
  }
}
