// All API response types live here. Never use `any` — add a proper type instead.
// Field names match the backend JSON exactly (snake_case) so no conversion layer is needed.

export type TransactionType = 'earn' | 'spend' | 'admin_grant' | 'expire';
export type UserRole = 'student' | 'partner' | 'admin';

// Profile returned by GET /api/v1/me
export interface Profile {
  id: string;
  email: string;
  name: string;
  student_id: string;
  balance: number;
}

// User stored in the Zustand auth store: Profile + role extracted from JWT claims.
export interface User extends Profile {
  role: UserRole;
}

export interface Transaction {
  id: string;
  amount: number;
  type: TransactionType;
  description: string;
  partner_id: string;
  created_at: string;
}

export interface Partner {
  id: string;
  name: string;
  address: string;
  max_spend_pct: number;
}

// Token pair returned by POST /api/v1/auth/login
export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

// Response from GET /api/v1/me/qr
export interface QRResponse {
  token: string;
}

// Stats returned by GET /api/v1/admin/stats
export interface Stats {
  total_students: number;
  total_points_issued: number;
  total_points_spent: number;
  active_partners: number;
}

// Generic success envelope: every API response is wrapped in { "data": ... }
export interface ApiResponse<T> {
  data: T;
}

// Shape of paginated transaction endpoints (both /me/transactions and /admin/transactions)
export interface PaginatedResponse<T> {
  transactions: T[];
  total: number;
}

// Error envelope: { "error": "..." }
export interface ApiError {
  error: string;
}

// Returned by POST /api/v1/partner/spend on success.
export interface SpendResult {
  status: string;
  spent: number;
  new_balance: number;
}

// Transaction record as seen by an administrator (includes user email).
export interface AdminTransaction {
  id: string;
  user_id: string;
  user_email: string;
  partner_id: string;
  amount: number;
  type: TransactionType;
  description: string;
  created_at: string;
}

// Student record as seen by an administrator (includes created_at).
export interface AdminStudent {
  id: string;
  email: string;
  name: string;
  student_id: string;
  balance: number;
  created_at: string;
}

// Paginated response from GET /api/v1/admin/transactions.
export interface AdminTransactionPage {
  transactions: AdminTransaction[];
  total: number;
}

// Paginated response from GET /api/v1/admin/users.
export interface AdminUsersPage {
  users: AdminStudent[];
  total: number;
}
