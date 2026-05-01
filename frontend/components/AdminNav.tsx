'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useAuthStore } from '@/lib/store';

const LINKS = [
  { href: '/admin/dashboard', label: 'Дашборд' },
  { href: '/admin/grant', label: 'Начислить' },
  { href: '/admin/transactions', label: 'Транзакции' },
  { href: '/admin/users', label: 'Студенты' },
] as const;

export function AdminNav() {
  const pathname = usePathname();
  const { logout } = useAuthStore();

  return (
    <nav className="border-b border-gray-700 bg-gray-900">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center gap-1 px-4 py-3">
        <span className="mr-4 text-sm font-semibold text-white">CU Points Admin</span>
        {LINKS.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            className={`rounded-lg px-3 py-1.5 text-sm transition-colors ${
              pathname === link.href
                ? 'bg-blue-600 text-white'
                : 'text-gray-400 hover:bg-gray-700 hover:text-white'
            }`}
          >
            {link.label}
          </Link>
        ))}
        <button
          onClick={logout}
          className="ml-auto rounded-lg px-3 py-1.5 text-sm text-gray-400 transition-colors hover:bg-gray-700 hover:text-white"
        >
          Выйти
        </button>
      </div>
    </nav>
  );
}
