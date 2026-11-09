/**
 * Cancel endpoint
 * POST /api/cancel/:command_id
 * 
 * Cancels a waiting command.
 */

import type { NextApiRequest, NextApiResponse } from 'next';

interface CancelResponse {
  ok: boolean;
}

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<CancelResponse | { detail: string }>
) {
  if (req.method !== 'POST') {
    res.setHeader('Allow', ['POST']);
    return res.status(405).json({ detail: 'Method Not Allowed' });
  }

  try {
    const { command_id } = req.query;
    
    if (!command_id || typeof command_id !== 'string') {
      return res.status(400).json({ detail: 'Command ID is required.' } as any);
    }

    // In production, this would:
    // 1. Check if the command is in a cancellable state (waiting)
    // 2. Set the cancel flag in Redis
    // 3. Return success
    
    // For now, return a mock success
    const response: CancelResponse = {
      ok: true,
    };

    return res.status(200).json(response);
  } catch (err) {
    console.error('Cancel error:', err);
    return res.status(500).json({ detail: 'Could not cancel that request.' } as any);
  }
}
