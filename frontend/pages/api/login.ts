/**
 * Login endpoint
 * GET /api/login
 * 
 * Authenticates the user and sets the session cookie.
 * In production, this would be called by the bot to generate a signed link.
 * Standard Node.js Vercel Function (Fluid Compute)
 */

import type { NextApiRequest, NextApiResponse } from 'next';
import { serialize } from 'cookie';

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<{ ok: boolean } | { detail: string }>
) {
  if (req.method !== 'GET') {
    res.setHeader('Allow', ['GET']);
    return res.status(405).json({ detail: 'Method Not Allowed' });
  }

  try {
    // In production, this would:
    // 1. Validate the one-time token from the bot
    // 2. Set the session cookie with the guild ID
    // 3. Redirect to the dashboard
    
    // For now, set a mock session cookie
    const guildId = req.query.guild_id as string || 'mock-guild-id';
    
    res.setHeader('Set-Cookie', serialize('session_guild_id', guildId, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'strict',
      maxAge: 60 * 60 * 12, // 12 hours
      path: '/',
    }));

    return res.status(200).json({ ok: true });
  } catch (err) {
    console.error('Login error:', err);
    return res.status(500).json({ detail: 'Could not authenticate.' });
  }
}
