/**
 * Session new endpoint
 * POST /api/session/new
 * 
 * Called when starting a new conversation.
 * Standard Node.js Vercel Function (Fluid Compute)
 */

import type { NextApiRequest, NextApiResponse } from 'next';

interface NewResponse {
  ok: boolean;
}

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<NewResponse | { detail: string }>
) {
  if (req.method !== 'POST') {
    res.setHeader('Allow', ['POST']);
    return res.status(405).json({ detail: 'Method Not Allowed' });
  }

  try {
    // In production, this would:
    // 1. Initialize a new session
    // 2. Assign an agent if available
    
    // For now, return a mock success
    const response: NewResponse = {
      ok: true,
    };

    return res.status(200).json(response);
  } catch (err) {
    console.error('New session error:', err);
    return res.status(500).json({ detail: 'Could not start new session.' });
  }
}
