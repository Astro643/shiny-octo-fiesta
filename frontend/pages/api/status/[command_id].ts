/**
 * Status endpoint
 * GET /api/status/:command_id
 * 
 * Returns the current status of a command.
 * Edge Function: Fast status lookup
 */

import type { NextApiRequest } from 'next';

interface StatusResponse {
  status: string;
  result?: string;
  error?: string;
}

// Mock status tracker - in production, this would query Redis
// Note: Edge functions have limited storage, so this is simplified
const mockStatuses: Record<string, StatusResponse> = {};

export const config = {
  runtime: 'edge', // Edge runtime for fast status lookup
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
    const { command_id } = req.query;
    
    if (!command_id || typeof command_id !== 'string') {
      return new Response(JSON.stringify({ detail: 'Command ID is required.' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      });
    }

    // In production, this would query Redis for the status
    // For now, generate a mock response
    // If we've seen this command before, return its status
    if (mockStatuses[command_id]) {
      return new Response(JSON.stringify(mockStatuses[command_id]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    }

    // Simulate processing
    const statuses = ['processing', 'waiting', 'completed', 'failed', 'rejected'];
    const randomStatus = statuses[Math.floor(Math.random() * statuses.length)];
    
    const response: StatusResponse = {
      status: randomStatus,
    };
    
    // Add result or error based on status
    if (randomStatus === 'completed') {
      response.result = 'Command executed successfully.';
    } else if (randomStatus === 'rejected') {
      response.error = 'Command rejected by audit.';
    } else if (randomStatus === 'failed') {
      response.error = 'Command failed to execute.';
    }

    // Store for future requests
    mockStatuses[command_id] = response;

    return new Response(JSON.stringify(response), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
  } catch (err) {
    console.error('Status error:', err);
    return new Response(JSON.stringify({ detail: 'Could not retrieve status.' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
