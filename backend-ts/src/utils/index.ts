/**
 * Utility functions for the Controller backend
 */

// Text utilities

/**
 * Clip a string to a maximum number of runes (Unicode characters)
 */
export function clipRunes(s: string, n: number): string {
  const runes = [...s];
  if (runes.length <= n) {
    return s;
  }
  return runes.slice(0, n).join('');
}

/**
 * Truncate a string to a maximum length
 */
export function truncateString(s: string, maxLen: number): string {
  if (s.length <= maxLen) {
    return s;
  }
  return s.slice(0, maxLen) + '...';
}

/**
 * Check if a string contains only digits
 */
export function isDigits(s: string): boolean {
  if (s === '') {
    return false;
  }
  return /^\d+$/.test(s);
}

/**
 * Check if a value is in a list
 */
export function inList(list: string[], s: string): boolean {
  return list.includes(s);
}

/**
 * Python-like string representation
 */
export function pyStr(v: any): string {
  if (v === null || v === undefined) {
    return 'None';
  }
  return String(v);
}

/**
 * JSON string without HTML escaping
 */
export function jsonString(v: any): string {
  return JSON.stringify(v);
}

/**
 * Get the first n items from an array
 */
export function head<T>(arr: T[], n: number): T[] {
  return arr.slice(0, n);
}

/**
 * Get the last n runes from a string
 */
export function tailRunes(s: string, n: number): string {
  const runes = [...s];
  if (runes.length <= n) {
    return s;
  }
  return runes.slice(-n).join('');
}

/**
 * Check if a value is truthy
 */
export function truthy(v: any): boolean {
  if (v === null || v === undefined) {
    return false;
  }
  if (typeof v === 'boolean') {
    return v;
  }
  if (typeof v === 'string') {
    return v !== '';
  }
  if (typeof v === 'number') {
    return v !== 0;
  }
  return true;
}

// Error handling utilities

/**
 * Classify an error into a category
 */
export function classifyError(err: Error): string {
  if (!err) {
    return '';
  }
  
  const message = err.message.toUpperCase();
  
  if (err.name === 'TimeoutError' || message.includes('TIMEOUT')) {
    return 'timeout';
  }
  
  if (message.includes('429') || message.includes('RESOURCE_EXHAUSTED') || message.includes('QUOTA')) {
    return 'rate_limited';
  }
  
  for (const code of ['500', '502', '503', '504', 'UNAVAILABLE', 'OVERLOADED']) {
    if (message.includes(code)) {
      return 'unavailable';
    }
  }
  
  return 'error';
}

/**
 * Check if a quota is spent (per-day or billing quota)
 */
export function quotaSpent(err: Error): boolean {
  const message = err.message.toUpperCase();
  return message.includes('PER DAY') || 
         message.includes('PERDAY') || 
         message.includes('BILLING') || 
         message.includes('DAILY');
}

// Time utilities

/**
 * Get current time in seconds since epoch
 */
export function nowSeconds(): number {
  return Date.now() / 1000;
}

/**
 * Sleep for a specified number of milliseconds
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// HTTP utilities

/**
 * Create an HTTP error
 */
export class HttpError extends Error {
  constructor(
    public status: number,
    public detail: string
  ) {
    super(detail);
    this.name = 'HttpError';
  }
}

/**
 * Create a new HTTP error
 */
export function newErr(status: number, detail: string): HttpError {
  return new HttpError(status, detail);
}

// Discord-specific utilities

/**
 * Check if a string is a valid Discord guild ID
 */
export function isGuildID(s: string): boolean {
  return /^\d{15,25}$/.test(s);
}

/**
 * Check if a string is a valid token
 */
export function isToken(s: string): boolean {
  return /^[A-Za-z0-9_-]{8,128}$/.test(s);
}
