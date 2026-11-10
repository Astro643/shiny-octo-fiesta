/**
 * Login endpoint
 * GET /api/login
 * 
 * Authenticates the user and sets the session cookie.
 * In production, this would be called by the bot to generate a signed link.
 * Edge Function: Fast authentication
 */

import type { NextApiRequest } from 'next';

export const config = {
  runtime: 'edge', // Edge runtime for fast authentication
};

export default function handler(
  req: NextApiRequest
) {
  if (req.method !== 'GET') {
    return new Response(JSON.stringify({ detail: 'Method Not Allowed' }), {
      status: 405,
      headers: { 'Allow': 'GET', 'Content-Type': 'application/json' },
    });
  }

  try {
    // In production, this would:
    // 1. Validate the one-time token from the bot
    // 2. Set the session cookie with the guild ID
    // 3. Redirect to the dashboard
    
    // For now, set a mock session cookie
    // Note: Edge functions cannot set cookies with Set-Cookie header directly
    // This is a limitation we need to work around
    const guildId = req.nextUrl.searchParams.get('guild_id') || 'mock-guild-id';
    
    // For Edge functions, we return the cookie in a custom header
    // The frontend can then set it using document.cookie
    return new Response(JSON.stringify({ ok: true, guildId }), {
      status: 200,
      headers: { 
        'Content-Type': 'application/json',
        'X-Session-Guild-Id': guildId,
      },
    });
  } catch (err) {
    console.error('Login error:', err);
    return new Response(JSON.stringify({ detail: 'Could not authenticate.' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
