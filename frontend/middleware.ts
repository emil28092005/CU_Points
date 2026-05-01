import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Decodes a JWT payload without verifying the signature.
// Safe for middleware because we only need the role for redirects — the backend
// will reject any tampered token when the actual API call is made.
function getJwtPayload(token: string): { role?: string; exp?: number } | null {
  try {
    const part = token.split('.')[1];
    if (!part) return null;
    // Fix base64url → base64 padding
    const padded = part.padEnd(part.length + ((4 - (part.length % 4)) % 4), '=');
    const decoded = atob(padded.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(decoded) as { role?: string; exp?: number };
  } catch {
    return null;
  }
}

// Maps URL path prefixes to the role required to access them.
const ROLE_REQUIREMENTS: [RegExp, string][] = [
  [/^\/(dashboard|history|qr|partners)/, 'student'],
  [/^\/scan/, 'partner'],
  [/^\/admin/, 'admin'],
];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  const required = ROLE_REQUIREMENTS.find(([re]) => re.test(pathname))?.[1];
  if (!required) return NextResponse.next();

  const token = request.cookies.get('access_token')?.value;
  if (!token) {
    return NextResponse.redirect(new URL('/login', request.url));
  }

  const payload = getJwtPayload(token);

  // Redirect if the token is missing, expired, or has the wrong role.
  if (!payload || payload.role !== required) {
    return NextResponse.redirect(new URL('/login', request.url));
  }
  if (payload.exp !== undefined && payload.exp * 1000 < Date.now()) {
    return NextResponse.redirect(new URL('/login', request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/(dashboard|history|qr|partners|scan|admin)(.*)'],
};
