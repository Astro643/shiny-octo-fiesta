/**
 * Health check endpoint
 * GET /api/healthz
 * 
 * Edge Function: Fast, simple health check
 */

import type { NextApiRequest, NextApiResponse } from 'next';

export const config = {
  runtime: 'edge', // Edge runtime for faster response
};

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  return new Response(JSON.stringify({ ok: true }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}
