/**
 * Health check endpoint
 * GET /api/healthz
 * 
 * Standard Node.js Vercel Function (Fluid Compute)
 */

import type { NextApiRequest, NextApiResponse } from 'next';

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  res.status(200).json({ ok: true });
}
