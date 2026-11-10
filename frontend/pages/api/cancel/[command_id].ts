/**
 * Cancel endpoint
 * POST /api/cancel/:command_id
 * 
 * Cancels a waiting command.
 * Edge Function: Fast cancellation operation
 */

import type { NextApiRequest } from 'next';

interface CancelResponse {
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
    const { command_id } = req.query;
    
    if (!command_id || typeof command_id !== 'string') {
      return new Response(JSON.stringify({ detail: 'Command ID is required.' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      });
    }

    // In production, this would:
    // 1. Check if the command is in a cancellable state (waiting)
    // 2. Set the cancel flag in Redis
    // 3. Return success
    
    // For now, return a mock success
    const response: CancelResponse = {
      ok: true,
    };

    return new Response(JSON.stringify(response), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
  } catch (err) {
    console.error('Cancel error:', err);
    return new Response(JSON.stringify({ detail: 'Could not cancel that request.' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
