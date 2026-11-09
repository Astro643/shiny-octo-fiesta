/**
 * Status endpoint
 * GET /api/status/:command_id
 * 
 * Returns the current status of a command.
 */

import type { NextApiRequest, NextApiResponse } from 'next';

interface StatusResponse {
  status: string;
  result?: string;
  error?: string;
}

// Mock status tracker - in production, this would query Redis
const mockStatuses: Record<string, StatusResponse> = {};

// Initialize with some mock data
setInterval(() => {
  // Clean up old entries
  Object.keys(mockStatuses).forEach(id => {
    if (Date.now() - mockStatuses[id].status.lastUpdated > 3600000) {
      delete mockStatuses[id];
    }
  });
}, 60000);

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<StatusResponse | { detail: string }>
) {
  if (req.method !== 'GET') {
    res.setHeader('Allow', ['GET']);
    return res.status(405).json({ detail: 'Method Not Allowed' });
  }

  try {
    const { command_id } = req.query;
    
    if (!command_id || typeof command_id !== 'string') {
      return res.status(400).json({ detail: 'Command ID is required.' } as any);
    }

    // In production, this would query Redis for the status
    // For now, generate a mock response
    // If we've seen this command before, return its status
    if (mockStatuses[command_id]) {
      return res.status(200).json(mockStatuses[command_id]);
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

    return res.status(200).json(response);
  } catch (err) {
    console.error('Status error:', err);
    return res.status(500).json({ detail: 'Could not retrieve status.' } as any);
  }
}
